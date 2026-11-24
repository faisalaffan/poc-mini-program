package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GachaRepository struct {
	pool *pgxpool.Pool
}

func NewGachaRepository(pool *pgxpool.Pool) *GachaRepository {
	return &GachaRepository{pool: pool}
}

func (r *GachaRepository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func ensureSyntheticUserTx(ctx context.Context, tx pgx.Tx, userID, displayName string, initialPoints int64) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "seed:"+userID); err != nil {
		return fmt.Errorf("advisory lock seed user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, display_name)
		VALUES ($1, $2)
		ON CONFLICT (id) DO NOTHING
	`, userID, displayName); err != nil {
		return fmt.Errorf("insert synthetic user: %w", err)
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(1) FROM point_ledger WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return fmt.Errorf("check user ledger seed: %w", err)
	}

	if count == 0 && initialPoints > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO point_ledger (user_id, entry_type, delta_points, reference_id)
			VALUES ($1, 'SEED', $2, $3)
		`, userID, initialPoints, "poc-initial-seed:"+userID); err != nil {
			return fmt.Errorf("insert initial seed ledger: %w", err)
		}
	}
	return nil
}

// EnsureSyntheticUser memastikan user sintetis POC ada dan memiliki saldo awal 1.000 poin jika belum punya entri ledger.
func (r *GachaRepository) EnsureSyntheticUser(ctx context.Context, userID, displayName string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin ensure user tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := ensureSyntheticUserTx(ctx, tx, userID, displayName, 1000); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *GachaRepository) GetBalance(ctx context.Context, userID string) (domain.Balance, error) {
	var bal domain.Balance
	bal.UserID = userID
	var updatedAt *time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(delta_points), 0)::bigint,
			COALESCE(MAX(id), 0)::bigint,
			MAX(created_at)
		FROM point_ledger
		WHERE user_id = $1
	`, userID).Scan(&bal.Balance, &bal.LastLedgerID, &updatedAt)
	if err != nil {
		return domain.Balance{}, fmt.Errorf("query balance: %w", err)
	}
	if updatedAt != nil {
		bal.UpdatedAt = updatedAt.UTC()
	} else {
		bal.UpdatedAt = time.Now().UTC()
	}
	return bal, nil
}

func (r *GachaRepository) GetActiveProbabilityTable(ctx context.Context) (domain.ProbabilityTable, error) {
	return getActiveProbabilityTableQuerier(ctx, r.pool)
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func getActiveProbabilityTableQuerier(ctx context.Context, q querier) (domain.ProbabilityTable, error) {
	var table domain.ProbabilityTable
	err := q.QueryRow(ctx, `
		SELECT version, draw_cost
		FROM probability_tables
		WHERE is_active = TRUE
		LIMIT 1
	`).Scan(&table.TableVersion, &table.DrawCost)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ProbabilityTable{}, domain.ErrNotFound
		}
		return domain.ProbabilityTable{}, fmt.Errorf("query active probability table: %w", err)
	}

	rows, err := q.Query(ctx, `
		SELECT id, table_version, code, name, rarity, reward_points, weight_bps, display_order
		FROM reward_tiers
		WHERE table_version = $1
		ORDER BY display_order ASC
	`, table.TableVersion)
	if err != nil {
		return domain.ProbabilityTable{}, fmt.Errorf("query reward tiers: %w", err)
	}
	defer rows.Close()

	totalBPS := 0
	for rows.Next() {
		var tier domain.RewardTier
		if err := rows.Scan(
			&tier.ID,
			&tier.TableVersion,
			&tier.Code,
			&tier.Name,
			&tier.Rarity,
			&tier.RewardPoints,
			&tier.WeightBPS,
			&tier.DisplayOrder,
		); err != nil {
			return domain.ProbabilityTable{}, fmt.Errorf("scan reward tier: %w", err)
		}
		tier.ProbabilityPercent = float64(tier.WeightBPS) / 100.0
		totalBPS += tier.WeightBPS
		table.Tiers = append(table.Tiers, tier)
	}
	if err := rows.Err(); err != nil {
		return domain.ProbabilityTable{}, fmt.Errorf("iterate reward tiers: %w", err)
	}
	table.TotalWeightBPS = totalBPS
	return table, nil
}

// ExecuteDraw menjalankan transaksi atomik draw dengan advisory lock, idempotency check, ledger append-only, dan audit log.
func (r *GachaRepository) ExecuteDraw(
	ctx context.Context,
	req domain.DrawRequest,
	pickTier func(domain.ProbabilityTable) (domain.RewardTier, error),
) (domain.DrawResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.DrawResult{}, fmt.Errorf("begin draw tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialize concurrent draw requests per user untuk mencegah race condition & double spend.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, req.UserID); err != nil {
		return domain.DrawResult{}, fmt.Errorf("acquire user advisory lock: %w", err)
	}

	existing, found, err := findExistingDrawTx(ctx, tx, req.UserID, req.IdempotencyKey)
	if err != nil {
		return domain.DrawResult{}, err
	}
	if found {
		if existing.requestHash != req.RequestHash {
			_ = insertAuditTx(ctx, tx, domain.AuditEvent{
				UserID:         req.UserID,
				DeviceID:       req.DeviceID,
				Action:         "DRAW",
				Outcome:        "REJECTED_IDEMPOTENCY_MISMATCH",
				IdempotencyKey: req.IdempotencyKey,
				ClientApproach: string(req.ClientApproach),
			})
			_ = tx.Commit(ctx)
			return domain.DrawResult{}, domain.ErrIdempotencyPayloadMismatch
		}
		existing.result.IdempotencyReplayed = true
		return existing.result, nil
	}

	table, err := getActiveProbabilityTableQuerier(ctx, tx)
	if err != nil {
		return domain.DrawResult{}, err
	}

	if req.ExpectedTableVersion != "" && req.ExpectedTableVersion != table.TableVersion {
		_ = insertAuditTx(ctx, tx, domain.AuditEvent{
			UserID:         req.UserID,
			DeviceID:       req.DeviceID,
			Action:         "DRAW",
			Outcome:        "REJECTED_TABLE_VERSION_MISMATCH",
			TableVersion:   table.TableVersion,
			IdempotencyKey: req.IdempotencyKey,
			ClientApproach: string(req.ClientApproach),
			Metadata: map[string]any{
				"expected_table_version": req.ExpectedTableVersion,
				"active_table_version":   table.TableVersion,
			},
		})
		_ = tx.Commit(ctx)
		return domain.DrawResult{}, domain.ErrTableVersionMismatch
	}

	var currentBalance int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(delta_points), 0)::bigint
		FROM point_ledger
		WHERE user_id = $1
	`, req.UserID).Scan(&currentBalance); err != nil {
		return domain.DrawResult{}, fmt.Errorf("query balance inside tx: %w", err)
	}

	if currentBalance < table.DrawCost {
		_ = insertAuditTx(ctx, tx, domain.AuditEvent{
			UserID:         req.UserID,
			DeviceID:       req.DeviceID,
			Action:         "DRAW",
			Outcome:        "REJECTED_INSUFFICIENT_BALANCE",
			TableVersion:   table.TableVersion,
			IdempotencyKey: req.IdempotencyKey,
			ClientApproach: string(req.ClientApproach),
			Metadata: map[string]any{
				"balance":   currentBalance,
				"draw_cost": table.DrawCost,
			},
		})
		_ = tx.Commit(ctx)
		return domain.DrawResult{}, domain.ErrInsufficientBalance
	}

	reward, err := pickTier(table)
	if err != nil {
		return domain.DrawResult{}, fmt.Errorf("pick reward tier: %w", err)
	}

	balanceAfter := currentBalance - table.DrawCost + reward.RewardPoints

	var drawID string
	var createdAt time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO gacha_draws (
			user_id, idempotency_key, request_hash, table_version,
			reward_tier_id, draw_cost, reward_points,
			balance_before, balance_after, client_approach, device_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id::text, created_at
	`,
		req.UserID,
		req.IdempotencyKey,
		req.RequestHash,
		table.TableVersion,
		reward.ID,
		table.DrawCost,
		reward.RewardPoints,
		currentBalance,
		balanceAfter,
		string(req.ClientApproach),
		req.DeviceID,
	).Scan(&drawID, &createdAt)
	if err != nil {
		return domain.DrawResult{}, fmt.Errorf("insert gacha_draws: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO point_ledger (user_id, entry_type, delta_points, reference_id)
		VALUES ($1, 'DRAW_DEBIT', $2, $3)
	`, req.UserID, -table.DrawCost, drawID); err != nil {
		return domain.DrawResult{}, fmt.Errorf("insert debit ledger: %w", err)
	}

	if reward.RewardPoints > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO point_ledger (user_id, entry_type, delta_points, reference_id)
			VALUES ($1, 'REWARD_CREDIT', $2, $3)
		`, req.UserID, reward.RewardPoints, drawID); err != nil {
			return domain.DrawResult{}, fmt.Errorf("insert credit ledger: %w", err)
		}
	}

	if err := insertAuditTx(ctx, tx, domain.AuditEvent{
		UserID:         req.UserID,
		DeviceID:       req.DeviceID,
		Action:         "DRAW",
		Outcome:        "SUCCESS",
		TableVersion:   table.TableVersion,
		DrawID:         &drawID,
		IdempotencyKey: req.IdempotencyKey,
		ClientApproach: string(req.ClientApproach),
		Metadata: map[string]any{
			"reward_code":    reward.Code,
			"reward_points":  reward.RewardPoints,
			"balance_before": currentBalance,
			"balance_after":  balanceAfter,
		},
	}); err != nil {
		return domain.DrawResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.DrawResult{}, fmt.Errorf("commit draw tx: %w", err)
	}

	return domain.DrawResult{
		DrawID:              drawID,
		IdempotencyKey:      req.IdempotencyKey,
		IdempotencyReplayed: false,
		TableVersion:        table.TableVersion,
		DrawCost:            table.DrawCost,
		Reward:              reward,
		BalanceBefore:       currentBalance,
		BalanceAfter:        balanceAfter,
		ClientApproach:      req.ClientApproach,
		CreatedAt:           createdAt.UTC(),
	}, nil
}

type existingDrawRow struct {
	requestHash string
	result      domain.DrawResult
}

func findExistingDrawTx(ctx context.Context, tx pgx.Tx, userID, idempotencyKey string) (existingDrawRow, bool, error) {
	var row existingDrawRow
	var tier domain.RewardTier
	var approach string
	var createdAt time.Time

	err := tx.QueryRow(ctx, `
		SELECT
			gd.id::text, gd.request_hash, gd.table_version, gd.draw_cost,
			gd.balance_before, gd.balance_after, gd.client_approach, gd.created_at,
			rt.id, rt.code, rt.name, rt.rarity, rt.reward_points, rt.weight_bps, rt.display_order
		FROM gacha_draws gd
		JOIN reward_tiers rt ON rt.id = gd.reward_tier_id
		WHERE gd.user_id = $1 AND gd.idempotency_key = $2
	`, userID, idempotencyKey).Scan(
		&row.result.DrawID,
		&row.requestHash,
		&row.result.TableVersion,
		&row.result.DrawCost,
		&row.result.BalanceBefore,
		&row.result.BalanceAfter,
		&approach,
		&createdAt,
		&tier.ID,
		&tier.Code,
		&tier.Name,
		&tier.Rarity,
		&tier.RewardPoints,
		&tier.WeightBPS,
		&tier.DisplayOrder,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return existingDrawRow{}, false, nil
		}
		return existingDrawRow{}, false, fmt.Errorf("query existing idempotency draw: %w", err)
	}

	tier.TableVersion = row.result.TableVersion
	tier.ProbabilityPercent = float64(tier.WeightBPS) / 100.0
	row.result.IdempotencyKey = idempotencyKey
	row.result.Reward = tier
	row.result.ClientApproach = domain.ParseClientApproach(approach)
	row.result.CreatedAt = createdAt.UTC()
	return row, true, nil
}

func (r *GachaRepository) GetHistory(ctx context.Context, userID string, limit int) ([]domain.HistoryItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			gd.id::text, gd.table_version, gd.draw_cost,
			rt.code, rt.name, rt.rarity, gd.reward_points,
			gd.balance_after, gd.client_approach, gd.created_at
		FROM gacha_draws gd
		JOIN reward_tiers rt ON rt.id = gd.reward_tier_id
		WHERE gd.user_id = $1
		ORDER BY gd.created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query draw history: %w", err)
	}
	defer rows.Close()

	items := make([]domain.HistoryItem, 0, limit)
	for rows.Next() {
		var item domain.HistoryItem
		var approach string
		var createdAt time.Time
		if err := rows.Scan(
			&item.DrawID,
			&item.TableVersion,
			&item.DrawCost,
			&item.RewardCode,
			&item.RewardName,
			&item.Rarity,
			&item.RewardPoints,
			&item.BalanceAfter,
			&approach,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan history item: %w", err)
		}
		item.ClientApproach = domain.ParseClientApproach(approach)
		item.CreatedAt = createdAt.UTC()
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *GachaRepository) AddSyntheticSeed(ctx context.Context, userID, deviceID string, points int64, refID string) (domain.Balance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Balance{}, fmt.Errorf("begin topup tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, userID); err != nil {
		return domain.Balance{}, fmt.Errorf("lock topup user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO point_ledger (user_id, entry_type, delta_points, reference_id)
		VALUES ($1, 'SEED', $2, $3)
	`, userID, points, refID); err != nil {
		return domain.Balance{}, fmt.Errorf("insert synthetic topup ledger: %w", err)
	}

	if err := insertAuditTx(ctx, tx, domain.AuditEvent{
		UserID:   userID,
		DeviceID: deviceID,
		Action:   "SYNTHETIC_TOPUP",
		Outcome:  "SUCCESS",
		Metadata: map[string]any{"points": points, "reference_id": refID},
	}); err != nil {
		return domain.Balance{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Balance{}, fmt.Errorf("commit topup tx: %w", err)
	}
	return r.GetBalance(ctx, userID)
}

func (r *GachaRepository) Reconcile(ctx context.Context, userID string) (domain.ReconcileReport, error) {
	var rep domain.ReconcileReport
	rep.UserID = userID

	if err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(delta_points), 0)::bigint,
			COALESCE(SUM(CASE WHEN entry_type = 'SEED' THEN delta_points ELSE 0 END), 0)::bigint
		FROM point_ledger
		WHERE user_id = $1
	`, userID).Scan(&rep.LedgerBalance, &rep.TotalSeeded); err != nil {
		return domain.ReconcileReport{}, fmt.Errorf("reconcile query ledger: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(reward_points), 0)::bigint
		FROM village_quiz_submissions
		WHERE user_id = $1 AND is_correct = TRUE
	`, userID).Scan(&rep.TotalQuizRewards); err != nil {
		return domain.ReconcileReport{}, fmt.Errorf("reconcile query quiz rewards: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(draw_cost), 0)::bigint,
			COALESCE(SUM(reward_points), 0)::bigint,
			COUNT(1)::int
		FROM gacha_draws
		WHERE user_id = $1
	`, userID).Scan(&rep.TotalDrawCost, &rep.TotalRewardPoints, &rep.TotalDraws); err != nil {
		return domain.ReconcileReport{}, fmt.Errorf("reconcile query draws: %w", err)
	}

	rep.ComputedBalance = rep.TotalSeeded + rep.TotalQuizRewards - rep.TotalDrawCost + rep.TotalRewardPoints
	rep.Balanced = rep.LedgerBalance == rep.ComputedBalance
	return rep, nil
}

func (r *GachaRepository) RecordAudit(ctx context.Context, ev domain.AuditEvent) error {
	metaBytes, err := marshalMetadata(ev.Metadata)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO audit_logs (
			user_id, device_id, action, outcome, table_version,
			draw_id, idempotency_key, client_approach, automation_flag, metadata
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7, ''), NULLIF($8, ''), $9, $10::jsonb)
	`,
		ev.UserID,
		ev.DeviceID,
		ev.Action,
		ev.Outcome,
		ev.TableVersion,
		ev.DrawID,
		ev.IdempotencyKey,
		ev.ClientApproach,
		ev.AutomationFlag,
		string(metaBytes),
	)
	return err
}

func insertAuditTx(ctx context.Context, tx pgx.Tx, ev domain.AuditEvent) error {
	metaBytes, err := marshalMetadata(ev.Metadata)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (
			user_id, device_id, action, outcome, table_version,
			draw_id, idempotency_key, client_approach, automation_flag, metadata
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7, ''), NULLIF($8, ''), $9, $10::jsonb)
	`,
		ev.UserID,
		ev.DeviceID,
		ev.Action,
		ev.Outcome,
		ev.TableVersion,
		ev.DrawID,
		ev.IdempotencyKey,
		ev.ClientApproach,
		ev.AutomationFlag,
		string(metaBytes),
	)
	if err != nil {
		return fmt.Errorf("insert audit_logs: %w", err)
	}
	return nil
}

func marshalMetadata(meta map[string]any) ([]byte, error) {
	if len(meta) == 0 {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal audit metadata: %w", err)
	}
	return b, nil
}
