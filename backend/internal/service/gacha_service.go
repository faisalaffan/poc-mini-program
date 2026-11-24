package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
	"github.com/google/uuid"
)

type GachaStore interface {
	Ping(ctx context.Context) error
	EnsureSyntheticUser(ctx context.Context, userID, displayName string) error
	GetBalance(ctx context.Context, userID string) (domain.Balance, error)
	GetActiveProbabilityTable(ctx context.Context) (domain.ProbabilityTable, error)
	ExecuteDraw(ctx context.Context, req domain.DrawRequest, pickTier func(domain.ProbabilityTable) (domain.RewardTier, error)) (domain.DrawResult, error)
	GetHistory(ctx context.Context, userID string, limit int) ([]domain.HistoryItem, error)
	AddSyntheticSeed(ctx context.Context, userID, deviceID string, points int64, refID string) (domain.Balance, error)
	Reconcile(ctx context.Context, userID string) (domain.ReconcileReport, error)
	RecordAudit(ctx context.Context, ev domain.AuditEvent) error
	GetVillageState(ctx context.Context, userID string) (domain.VillageState, error)
	SubmitQuiz(ctx context.Context, req domain.QuizSubmitRequest) (domain.QuizSubmitResult, error)
	ResetUserQuizzes(ctx context.Context, userID string) (int64, error)
	GetSDUIScreen(ctx context.Context, screenID, variant string) (domain.SDUIScreen, error)
	UpsertSDUIScreen(ctx context.Context, screen domain.SDUIScreen) (domain.SDUIScreen, error)
}

type GachaService struct {
	store  GachaStore
	rng    WeightedRNG
	appEnv string
}

func NewGachaService(store GachaStore, rng WeightedRNG, appEnv string) *GachaService {
	return &GachaService{
		store:  store,
		rng:    rng,
		appEnv: appEnv,
	}
}

func (s *GachaService) HealthPing(ctx context.Context) error {
	return s.store.Ping(ctx)
}

func (s *GachaService) EnsureUser(ctx context.Context, userID string) error {
	cleanID := strings.TrimSpace(userID)
	if cleanID == "" || len(cleanID) > 64 {
		return domain.ErrInvalidInput
	}
	return s.store.EnsureSyntheticUser(ctx, cleanID, "Nasabah POC "+cleanID)
}

func (s *GachaService) GetBalance(ctx context.Context, userID string) (domain.Balance, error) {
	return s.store.GetBalance(ctx, userID)
}

func (s *GachaService) GetProbabilities(ctx context.Context) (domain.ProbabilityTable, error) {
	return s.store.GetActiveProbabilityTable(ctx)
}

func (s *GachaService) ExecuteDraw(ctx context.Context, req domain.DrawRequest) (domain.DrawResult, error) {
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return domain.DrawResult{}, domain.ErrMissingIdempotencyKey
	}
	if len(req.IdempotencyKey) > 128 {
		return domain.DrawResult{}, domain.ErrInvalidInput
	}
	if strings.TrimSpace(req.DeviceID) == "" {
		return domain.DrawResult{}, domain.ErrMissingDeviceID
	}

	req.RequestHash = hashDrawPayload(req.UserID, req.ExpectedTableVersion)
	return s.store.ExecuteDraw(ctx, req, func(table domain.ProbabilityTable) (domain.RewardTier, error) {
		return s.rng.Pick(table.Tiers)
	})
}

func (s *GachaService) GetHistory(ctx context.Context, userID string, limit int) ([]domain.HistoryItem, error) {
	return s.store.GetHistory(ctx, userID, limit)
}

func (s *GachaService) SyntheticTopUp(ctx context.Context, userID, deviceID string, points int64) (domain.Balance, error) {
	if s.appEnv != "poc" {
		return domain.Balance{}, domain.ErrPOCOnly
	}
	if points <= 0 || points > 100000 {
		return domain.Balance{}, domain.ErrInvalidInput
	}
	refID := "poc-topup:" + uuid.NewString()
	return s.store.AddSyntheticSeed(ctx, userID, deviceID, points, refID)
}

func (s *GachaService) Reconcile(ctx context.Context, userID string) (domain.ReconcileReport, error) {
	return s.store.Reconcile(ctx, userID)
}

func (s *GachaService) RecordSecurityAudit(ctx context.Context, ev domain.AuditEvent) {
	_ = s.store.RecordAudit(ctx, ev)
}

func (s *GachaService) GetVillageState(ctx context.Context, userID string) (domain.VillageState, error) {
	return s.store.GetVillageState(ctx, userID)
}

func (s *GachaService) ResetUserQuizzes(ctx context.Context, userID string) (int64, error) {
	return s.store.ResetUserQuizzes(ctx, userID)
}

func (s *GachaService) SubmitQuiz(ctx context.Context, req domain.QuizSubmitRequest) (domain.QuizSubmitResult, error) {
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return domain.QuizSubmitResult{}, domain.ErrMissingIdempotencyKey
	}
	if len(req.IdempotencyKey) > 128 {
		return domain.QuizSubmitResult{}, domain.ErrInvalidInput
	}
	if strings.TrimSpace(req.DeviceID) == "" {
		return domain.QuizSubmitResult{}, domain.ErrMissingDeviceID
	}
	req.RequestHash = hashQuizPayload(req.UserID, req.QuizID, req.SelectedIndex)
	return s.store.SubmitQuiz(ctx, req)
}

func hashDrawPayload(userID, expectedTableVersion string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s", userID, expectedTableVersion)))
	return hex.EncodeToString(sum[:])
}

func hashQuizPayload(userID, quizID string, selectedIndex int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d", userID, quizID, selectedIndex)))
	return hex.EncodeToString(sum[:])
}
