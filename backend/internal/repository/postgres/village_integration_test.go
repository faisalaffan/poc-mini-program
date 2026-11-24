package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
	"github.com/faisalaffan/poc-mini-program/backend/internal/service"
	"github.com/google/uuid"
)

func TestIntegration_VillageQuizIdempotencyAndReconcile(t *testing.T) {
	pool, repo := setupIntegrationDB(t)
	userID := seedUserWithExactPoints(t, pool, 1000)
	svc := service.NewGachaService(repo, service.NewCryptoWeightedRNG(), "poc")
	ctx := context.Background()

	state, err := svc.GetVillageState(ctx, userID)
	if err != nil {
		t.Fatalf("get village state: %v", err)
	}
	if len(state.Quizzes) != 3 {
		t.Fatalf("expected 3 seeded quizzes, got %d", len(state.Quizzes))
	}

	// Wrong answer gives 0 reward and allows retry
	wrongRes, err := svc.SubmitQuiz(ctx, domain.QuizSubmitRequest{
		UserID:         userID,
		DeviceID:       "sim-quiz-1",
		QuizID:         "quiz_tower",
		SelectedIndex:  0,
		IdempotencyKey: uuid.NewString(),
		ClientApproach: domain.ApproachSDUI,
	})
	if err != nil {
		t.Fatalf("submit wrong answer: %v", err)
	}
	if wrongRes.Correct || wrongRes.RewardPoints != 0 || wrongRes.BalanceAfter != 1000 {
		t.Fatalf("unexpected wrong answer result: %+v", wrongRes)
	}

	// 8 concurrent retries with SAME Idempotency-Key for correct answer (index 1 on quiz_tower -> +150)
	const workers = 8
	sharedKey := "quiz-idem-" + uuid.NewString()
	var wg sync.WaitGroup
	results := make([]domain.QuizSubmitResult, workers)
	errs := make([]error, workers)
	for i := range workers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = svc.SubmitQuiz(ctx, domain.QuizSubmitRequest{
				UserID:         userID,
				DeviceID:       "sim-quiz-1",
				QuizID:         "quiz_tower",
				SelectedIndex:  1,
				IdempotencyKey: sharedKey,
				ClientApproach: domain.ApproachNative,
			})
		}(i)
	}
	wg.Wait()

	fresh := 0
	replayed := 0
	for i := range workers {
		if errs[i] != nil {
			t.Fatalf("worker %d error: %v", i, errs[i])
		}
		if results[i].BalanceAfter != 1150 || results[i].RewardPoints != 150 || !results[i].Correct {
			t.Fatalf("unexpected quiz result: %+v", results[i])
		}
		if results[i].IdempotencyReplayed {
			replayed++
		} else {
			fresh++
		}
	}
	if fresh != 1 || replayed != workers-1 {
		t.Fatalf("expected 1 fresh & %d replayed, got %d & %d", workers-1, fresh, replayed)
	}

	// New submission with different key for already-completed quiz must fail
	if _, err := svc.SubmitQuiz(ctx, domain.QuizSubmitRequest{
		UserID:         userID,
		DeviceID:       "sim-quiz-1",
		QuizID:         "quiz_tower",
		SelectedIndex:  1,
		IdempotencyKey: uuid.NewString(),
		ClientApproach: domain.ApproachWebView,
	}); !errors.Is(err, domain.ErrQuizAlreadyCompleted) {
		t.Fatalf("expected ErrQuizAlreadyCompleted, got %v", err)
	}

	// Gate 2 Reconcile must include quiz rewards and remain balanced
	rep, err := svc.Reconcile(ctx, userID)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if !rep.Balanced || rep.LedgerBalance != 1150 || rep.TotalQuizRewards != 150 {
		t.Fatalf("unexpected reconcile report: %+v", rep)
	}
}

func TestIntegration_SDUIScreenStoredInDBAndUpdatable(t *testing.T) {
	pool, repo := setupIntegrationDB(t)
	userID := seedUserWithExactPoints(t, pool, 1000)
	sduiSvc := service.NewSDUIService(repo)
	ctx := context.Background()

	screen, err := sduiSvc.BuildVillageScreen(ctx, userID, "default", 0)
	if err != nil {
		t.Fatalf("build village screen from DB: %v", err)
	}
	if screen.ScreenID != "livin_village_home" || len(screen.Components) < 6 {
		t.Fatalf("unexpected seeded SDUI screen: %+v", screen)
	}

	// Tambah komponen button baru ke variant 'custom-test' di DB lalu baca kembali
	custom := screen
	custom.Variant = "custom-test"
	openRewards := domain.SDUIActionOpenRewards
	custom.Components = append(custom.Components, domain.SDUINode{
		ID:     "fab_extra_test",
		Type:   domain.SDUITypeFloatingAction,
		Props:  map[string]any{"icon": "gift", "color": "blue", "label": "Bonus"},
		Action: &openRewards,
	})
	if _, err := sduiSvc.UpdateVillageScreen(ctx, custom); err != nil {
		t.Fatalf("upsert custom sdui screen: %v", err)
	}

	loaded, err := sduiSvc.BuildVillageScreen(ctx, userID, "custom-test", 0)
	if err != nil {
		t.Fatalf("load custom sdui screen: %v", err)
	}
	if len(loaded.Components) != len(screen.Components)+1 {
		t.Fatalf("expected %d components after DB update, got %d", len(screen.Components)+1, len(loaded.Components))
	}
}
