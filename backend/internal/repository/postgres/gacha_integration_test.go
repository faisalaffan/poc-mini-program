package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
	"github.com/faisalaffan/poc-mini-program/backend/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupIntegrationDB(t *testing.T) (*pgxpool.Pool, *GachaRepository) {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://gacha:gacha_poc_password@127.0.0.1:5432/gacha_poc?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect test postgres (%s): %v", dbURL, err)
	}
	t.Cleanup(pool.Close)

	if err := MigrateAndSeed(ctx, pool); err != nil {
		t.Fatalf("failed to migrate and seed test postgres: %v", err)
	}

	return pool, NewGachaRepository(pool)
}

func seedUserWithExactPoints(t *testing.T, pool *pgxpool.Pool, points int64) string {
	t.Helper()
	ctx := context.Background()
	userID := "test-user-" + uuid.NewString()

	if _, err := pool.Exec(ctx, `INSERT INTO users (id, display_name) VALUES ($1, $2)`, userID, "Test User"); err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO point_ledger (user_id, entry_type, delta_points, reference_id)
		VALUES ($1, 'SEED', $2, $3)
	`, userID, points, "test-seed:"+userID); err != nil {
		t.Fatalf("insert seed ledger: %v", err)
	}
	return userID
}

type deterministicCommonRNG struct{}

func (deterministicCommonRNG) Pick(tiers []domain.RewardTier) (domain.RewardTier, error) {
	return tiers[0], nil // PTS_50 (50 poin)
}

func TestIntegration_IdempotencyConcurrentSameKey(t *testing.T) {
	pool, repo := setupIntegrationDB(t)
	userID := seedUserWithExactPoints(t, pool, 1000)
	svc := service.NewGachaService(repo, service.NewCryptoWeightedRNG(), "poc")

	const workers = 10
	sharedKey := "idem-" + uuid.NewString()

	var wg sync.WaitGroup
	results := make([]domain.DrawResult, workers)
	errs := make([]error, workers)

	for i := range workers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			res, err := svc.ExecuteDraw(context.Background(), domain.DrawRequest{
				UserID:               userID,
				DeviceID:             "sim-device-1",
				IdempotencyKey:       sharedKey,
				ExpectedTableVersion: "v1.0.0",
				ClientApproach:       domain.ApproachNative,
			})
			results[idx] = res
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	freshCount := 0
	replayCount := 0
	var expectedDrawID string
	for i := range workers {
		if errs[i] != nil {
			t.Fatalf("worker %d unexpected error: %v", i, errs[i])
		}
		if expectedDrawID == "" {
			expectedDrawID = results[i].DrawID
		} else if results[i].DrawID != expectedDrawID {
			t.Fatalf("draw_id mismatch across idempotent retries: %s vs %s", expectedDrawID, results[i].DrawID)
		}
		if results[i].IdempotencyReplayed {
			replayCount++
		} else {
			freshCount++
		}
	}

	if freshCount != 1 || replayCount != workers-1 {
		t.Fatalf("expected 1 fresh and %d replayed, got %d fresh and %d replayed", workers-1, freshCount, replayCount)
	}

	rep, err := svc.Reconcile(context.Background(), userID)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}
	if !rep.Balanced || rep.TotalDraws != 1 {
		t.Fatalf("unexpected reconcile report: %+v", rep)
	}
}

func TestIntegration_PreventDoubleSpendConcurrentDistinctKeys(t *testing.T) {
	pool, repo := setupIntegrationDB(t)
	// Modal tepat 100 poin & reward selalu 50 poin -> saldo setelah 1 draw = 50 (< 100), jadi hanya 1 draw boleh lolos.
	userID := seedUserWithExactPoints(t, pool, 100)
	svc := service.NewGachaService(repo, deterministicCommonRNG{}, "poc")

	const workers = 10
	var wg sync.WaitGroup
	errs := make([]error, workers)
	successResults := make([]domain.DrawResult, workers)

	for i := range workers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			res, err := svc.ExecuteDraw(context.Background(), domain.DrawRequest{
				UserID:               userID,
				DeviceID:             "sim-device-race",
				IdempotencyKey:       "race-key-" + uuid.NewString(),
				ExpectedTableVersion: "v1.0.0",
				ClientApproach:       domain.ApproachWebView,
			})
			successResults[idx] = res
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	successes := 0
	insufficient := 0
	for i := range workers {
		if errs[i] == nil {
			successes++
		} else if errors.Is(errs[i], domain.ErrInsufficientBalance) {
			insufficient++
		} else {
			t.Fatalf("worker %d unexpected error: %v", i, errs[i])
		}
	}

	if successes != 1 || insufficient != workers-1 {
		t.Fatalf("expected 1 success and %d insufficient, got %d successes and %d insufficient",
			workers-1, successes, insufficient)
	}

	bal, err := svc.GetBalance(context.Background(), userID)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if bal.Balance != 50 {
		t.Fatalf("expected final balance 50, got %d", bal.Balance)
	}

	rep, err := svc.Reconcile(context.Background(), userID)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if !rep.Balanced || rep.LedgerBalance != 50 || rep.TotalDraws != 1 {
		t.Fatalf("unexpected reconcile report: %+v", rep)
	}
}

func TestIntegration_AppendOnlyTriggerBlocksMutation(t *testing.T) {
	pool, repo := setupIntegrationDB(t)
	userID := seedUserWithExactPoints(t, pool, 500)
	svc := service.NewGachaService(repo, deterministicCommonRNG{}, "poc")

	if _, err := svc.ExecuteDraw(context.Background(), domain.DrawRequest{
		UserID:               userID,
		DeviceID:             "dev-audit",
		IdempotencyKey:       uuid.NewString(),
		ExpectedTableVersion: "v1.0.0",
		ClientApproach:       domain.ApproachSDUI,
	}); err != nil {
		t.Fatalf("execute draw: %v", err)
	}

	ctx := context.Background()
	queries := []string{
		`UPDATE point_ledger SET delta_points = 99999 WHERE user_id = $1`,
		`DELETE FROM point_ledger WHERE user_id = $1`,
		`UPDATE audit_logs SET outcome = 'TAMPERED' WHERE user_id = $1`,
		`DELETE FROM audit_logs WHERE user_id = $1`,
	}
	for _, q := range queries {
		if _, err := pool.Exec(ctx, q, userID); err == nil {
			t.Fatalf("expected append-only trigger error for query: %s", q)
		}
	}
}

func TestIntegration_TableVersionAndIdempotencyMismatch(t *testing.T) {
	pool, repo := setupIntegrationDB(t)
	userID := seedUserWithExactPoints(t, pool, 500)
	svc := service.NewGachaService(repo, deterministicCommonRNG{}, "poc")

	if _, err := svc.ExecuteDraw(context.Background(), domain.DrawRequest{
		UserID:               userID,
		DeviceID:             "dev-ver",
		IdempotencyKey:       uuid.NewString(),
		ExpectedTableVersion: "v0.9.0",
		ClientApproach:       domain.ApproachNative,
	}); !errors.Is(err, domain.ErrTableVersionMismatch) {
		t.Fatalf("expected ErrTableVersionMismatch, got %v", err)
	}

	idemKey := "mismatch-" + uuid.NewString()
	if _, err := svc.ExecuteDraw(context.Background(), domain.DrawRequest{
		UserID:               userID,
		DeviceID:             "dev-ver",
		IdempotencyKey:       idemKey,
		ExpectedTableVersion: "v1.0.0",
		ClientApproach:       domain.ApproachNative,
	}); err != nil {
		t.Fatalf("first draw failed: %v", err)
	}

	if _, err := svc.ExecuteDraw(context.Background(), domain.DrawRequest{
		UserID:               userID,
		DeviceID:             "dev-ver",
		IdempotencyKey:       idemKey,
		ExpectedTableVersion: "",
		ClientApproach:       domain.ApproachNative,
	}); !errors.Is(err, domain.ErrIdempotencyPayloadMismatch) {
		t.Fatalf("expected ErrIdempotencyPayloadMismatch, got %v", err)
	}
}
