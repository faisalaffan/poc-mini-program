package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

func (r *GachaRepository) GetVillageState(ctx context.Context, userID string) (domain.VillageState, error) {
	bal, err := r.GetBalance(ctx, userID)
	if err != nil {
		return domain.VillageState{}, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			vq.id, vq.building_name, vq.title, vq.pin_color, vq.x_ratio, vq.y_ratio,
			vq.reward_points, vq.question, vq.options, vq.correct_index, vq.display_order,
			EXISTS (
				SELECT 1 FROM village_quiz_submissions vqs
				WHERE vqs.user_id = $1 AND vqs.quiz_id = vq.id AND vqs.is_correct = TRUE
			) AS completed
		FROM village_quizzes vq
		ORDER BY vq.display_order ASC
	`, userID)
	if err != nil {
		return domain.VillageState{}, fmt.Errorf("query village quizzes: %w", err)
	}
	defer rows.Close()

	quizzes := make([]domain.VillageQuiz, 0, 3)
	completedCount := 0
	for rows.Next() {
		var q domain.VillageQuiz
		var rawColor string
		var rawOptions []byte
		if err := rows.Scan(
			&q.ID,
			&q.BuildingName,
			&q.Title,
			&rawColor,
			&q.XRatio,
			&q.YRatio,
			&q.RewardPoints,
			&q.Question,
			&rawOptions,
			&q.CorrectIndex,
			&q.DisplayOrder,
			&q.Completed,
		); err != nil {
			return domain.VillageState{}, fmt.Errorf("scan village quiz: %w", err)
		}
		q.PinColor = domain.PinColor(rawColor)
		if err := json.Unmarshal(rawOptions, &q.Options); err != nil {
			return domain.VillageState{}, fmt.Errorf("unmarshal quiz options: %w", err)
		}
		if q.Completed {
			completedCount++
		}
		quizzes = append(quizzes, q)
	}
	if err := rows.Err(); err != nil {
		return domain.VillageState{}, fmt.Errorf("iterate village quizzes: %w", err)
	}

	return domain.VillageState{
		UserID:         userID,
		CampaignBanner: "Mainkan kuis Livin' Village & dapatkan hadiah menarik pada 1 Jun 2026 - 31 Dec 2026",
		Balance:        bal.Balance,
		CompletedCount: completedCount,
		TotalQuizzes:   len(quizzes),
		Quizzes:        quizzes,
		Rules: []string{
			"Ketuk 3 pin kuis di atas gedung Livin' Village untuk menjawab pertanyaan.",
			"Setiap gedung hanya memberi hadiah poin satu kali setelah jawaban benar.",
			"Setiap pengajuan jawaban memakai Idempotency-Key & dicatat di ledger append-only.",
			"Ketuk ikon kado biru untuk melihat saldo poin & menukar kotak hadiah (100 poin).",
		},
	}, nil
}

func (r *GachaRepository) ResetUserQuizzes(ctx context.Context, userID string) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM village_quiz_submissions WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("reset user quiz submissions: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *GachaRepository) SubmitQuiz(ctx context.Context, req domain.QuizSubmitRequest) (domain.QuizSubmitResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.QuizSubmitResult{}, fmt.Errorf("begin quiz tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, req.UserID); err != nil {
		return domain.QuizSubmitResult{}, fmt.Errorf("acquire user advisory lock: %w", err)
	}

	var existingID string
	var existingHash string
	var existingCorrect bool
	var existingPoints int64
	var existingBefore int64
	var existingAfter int64
	var existingApproach string
	var existingCreated time.Time

	err = tx.QueryRow(ctx, `
		SELECT
			id::text, request_hash, is_correct, reward_points,
			balance_before, balance_after, client_approach, created_at
		FROM village_quiz_submissions
		WHERE user_id = $1 AND idempotency_key = $2
	`, req.UserID, req.IdempotencyKey).Scan(
		&existingID,
		&existingHash,
		&existingCorrect,
		&existingPoints,
		&existingBefore,
		&existingAfter,
		&existingApproach,
		&existingCreated,
	)
	if err == nil {
		if existingHash != req.RequestHash {
			return domain.QuizSubmitResult{}, domain.ErrIdempotencyPayloadMismatch
		}
		var explanation string
		_ = tx.QueryRow(ctx, `SELECT explanation FROM village_quizzes WHERE id = $1`, req.QuizID).Scan(&explanation)
		if !existingCorrect {
			explanation = "Jawaban belum tepat, silakan coba kembali."
		}
		return domain.QuizSubmitResult{
			SubmissionID:        existingID,
			QuizID:              req.QuizID,
			IdempotencyKey:      req.IdempotencyKey,
			IdempotencyReplayed: true,
			Correct:             existingCorrect,
			RewardPoints:        existingPoints,
			BalanceBefore:       existingBefore,
			BalanceAfter:        existingAfter,
			Explanation:         explanation,
			ClientApproach:      domain.ParseClientApproach(existingApproach),
			CreatedAt:           existingCreated.UTC(),
		}, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return domain.QuizSubmitResult{}, fmt.Errorf("query existing quiz submission: %w", err)
	}

	var rewardPoints int64
	var correctIndex int
	var optionsRaw []byte
	var explanation string
	err = tx.QueryRow(ctx, `
		SELECT reward_points, correct_index, options, explanation
		FROM village_quizzes
		WHERE id = $1
	`, req.QuizID).Scan(&rewardPoints, &correctIndex, &optionsRaw, &explanation)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.QuizSubmitResult{}, domain.ErrNotFound
		}
		return domain.QuizSubmitResult{}, fmt.Errorf("query quiz definition: %w", err)
	}

	var options []string
	if err := json.Unmarshal(optionsRaw, &options); err != nil {
		return domain.QuizSubmitResult{}, fmt.Errorf("unmarshal quiz options: %w", err)
	}
	if req.SelectedIndex < 0 || req.SelectedIndex >= len(options) {
		return domain.QuizSubmitResult{}, domain.ErrInvalidInput
	}

	var alreadyCompleted bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM village_quiz_submissions
			WHERE user_id = $1 AND quiz_id = $2 AND is_correct = TRUE
		)
	`, req.UserID, req.QuizID).Scan(&alreadyCompleted); err != nil {
		return domain.QuizSubmitResult{}, fmt.Errorf("check completed quiz: %w", err)
	}
	if alreadyCompleted {
		return domain.QuizSubmitResult{}, domain.ErrQuizAlreadyCompleted
	}

	var currentBalance int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(delta_points), 0)::bigint
		FROM point_ledger
		WHERE user_id = $1
	`, req.UserID).Scan(&currentBalance); err != nil {
		return domain.QuizSubmitResult{}, fmt.Errorf("query balance inside quiz tx: %w", err)
	}

	isCorrect := req.SelectedIndex == correctIndex
	granted := int64(0)
	msg := "Jawaban belum tepat, coba pilih jawaban lainnya."
	if isCorrect {
		granted = rewardPoints
		msg = explanation
	}
	balanceAfter := currentBalance + granted

	var submissionID string
	var createdAt time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO village_quiz_submissions (
			user_id, quiz_id, idempotency_key, request_hash,
			selected_index, is_correct, reward_points,
			balance_before, balance_after, client_approach, device_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id::text, created_at
	`,
		req.UserID,
		req.QuizID,
		req.IdempotencyKey,
		req.RequestHash,
		req.SelectedIndex,
		isCorrect,
		granted,
		currentBalance,
		balanceAfter,
		string(req.ClientApproach),
		req.DeviceID,
	).Scan(&submissionID, &createdAt)
	if err != nil {
		return domain.QuizSubmitResult{}, fmt.Errorf("insert quiz submission: %w", err)
	}

	if isCorrect && granted > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO point_ledger (user_id, entry_type, delta_points, reference_id)
			VALUES ($1, 'QUIZ_REWARD', $2, $3)
		`, req.UserID, granted, submissionID); err != nil {
			return domain.QuizSubmitResult{}, fmt.Errorf("insert quiz reward ledger: %w", err)
		}
	}

	outcome := "WRONG_ANSWER"
	if isCorrect {
		outcome = "SUCCESS"
	}
	if err := insertAuditTx(ctx, tx, domain.AuditEvent{
		UserID:         req.UserID,
		DeviceID:       req.DeviceID,
		Action:         "QUIZ_SUBMIT",
		Outcome:        outcome,
		IdempotencyKey: req.IdempotencyKey,
		ClientApproach: string(req.ClientApproach),
		Metadata: map[string]any{
			"quiz_id":        req.QuizID,
			"selected_index": req.SelectedIndex,
			"reward_points":  granted,
			"balance_after":  balanceAfter,
		},
	}); err != nil {
		return domain.QuizSubmitResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.QuizSubmitResult{}, fmt.Errorf("commit quiz tx: %w", err)
	}

	return domain.QuizSubmitResult{
		SubmissionID:        submissionID,
		QuizID:              req.QuizID,
		IdempotencyKey:      req.IdempotencyKey,
		IdempotencyReplayed: false,
		Correct:             isCorrect,
		RewardPoints:        granted,
		BalanceBefore:       currentBalance,
		BalanceAfter:        balanceAfter,
		Explanation:         msg,
		ClientApproach:      req.ClientApproach,
		CreatedAt:           createdAt.UTC(),
	}, nil
}
