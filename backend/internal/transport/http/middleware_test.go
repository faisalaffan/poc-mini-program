package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/faisalaffan/poc-mini-program/backend/internal/config"
	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
	"github.com/faisalaffan/poc-mini-program/backend/internal/service"
)

type stubStore struct{}

func (s *stubStore) Ping(context.Context) error { return nil }
func (s *stubStore) EnsureSyntheticUser(context.Context, string, string) error {
	return nil
}
func (s *stubStore) GetBalance(_ context.Context, userID string) (domain.Balance, error) {
	return domain.Balance{UserID: userID, Balance: 1000, LastLedgerID: 1, UpdatedAt: time.Now().UTC()}, nil
}
func (s *stubStore) GetActiveProbabilityTable(context.Context) (domain.ProbabilityTable, error) {
	return domain.ProbabilityTable{
		TableVersion:   "v1.0.0",
		DrawCost:       100,
		TotalWeightBPS: 10000,
		Tiers: []domain.RewardTier{
			{ID: "v1_common", Code: "PTS_50", Name: "50 Poin", Rarity: domain.RarityCommon, RewardPoints: 50, WeightBPS: 10000, ProbabilityPercent: 100, DisplayOrder: 1},
		},
	}, nil
}
func (s *stubStore) ExecuteDraw(_ context.Context, req domain.DrawRequest, _ func(domain.ProbabilityTable) (domain.RewardTier, error)) (domain.DrawResult, error) {
	return domain.DrawResult{
		DrawID:         "00000000-0000-0000-0000-000000000001",
		IdempotencyKey: req.IdempotencyKey,
		TableVersion:   "v1.0.0",
		DrawCost:       100,
		Reward:         domain.RewardTier{ID: "v1_common", Code: "PTS_50", Name: "50 Poin", Rarity: domain.RarityCommon, RewardPoints: 50, WeightBPS: 10000, ProbabilityPercent: 100, DisplayOrder: 1},
		BalanceBefore:  1000,
		BalanceAfter:   950,
		ClientApproach: req.ClientApproach,
		CreatedAt:      time.Now().UTC(),
	}, nil
}
func (s *stubStore) GetHistory(context.Context, string, int) ([]domain.HistoryItem, error) {
	return nil, nil
}
func (s *stubStore) AddSyntheticSeed(_ context.Context, userID, _ string, points int64, _ string) (domain.Balance, error) {
	return domain.Balance{UserID: userID, Balance: 1000 + points}, nil
}
func (s *stubStore) Reconcile(_ context.Context, userID string) (domain.ReconcileReport, error) {
	return domain.ReconcileReport{UserID: userID, Balanced: true}, nil
}
func (s *stubStore) RecordAudit(context.Context, domain.AuditEvent) error { return nil }
func (s *stubStore) GetVillageState(_ context.Context, userID string) (domain.VillageState, error) {
	return domain.VillageState{UserID: userID, Balance: 1000}, nil
}
func (s *stubStore) ResetUserQuizzes(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
func (s *stubStore) SubmitQuiz(_ context.Context, req domain.QuizSubmitRequest) (domain.QuizSubmitResult, error) {
	return domain.QuizSubmitResult{QuizID: req.QuizID, IdempotencyKey: req.IdempotencyKey, Correct: true, RewardPoints: 150, BalanceBefore: 1000, BalanceAfter: 1150}, nil
}
func (s *stubStore) GetSDUIScreen(_ context.Context, screenID, variant string) (domain.SDUIScreen, error) {
	return domain.SDUIScreen{SchemaVersion: 1, ScreenID: screenID, Variant: variant}, nil
}
func (s *stubStore) UpsertSDUIScreen(_ context.Context, screen domain.SDUIScreen) (domain.SDUIScreen, error) {
	return screen, nil
}

func newTestRouter(t *testing.T, rl *RateLimiter) (http.Handler, *service.AuthService) {
	t.Helper()
	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<!doctype html><html><body>Vue SPA</body></html>"), 0o600); err != nil {
		t.Fatalf("write temp index.html: %v", err)
	}

	cfg := config.Config{
		Port:                "8080",
		AppEnv:              "poc",
		BuildVersion:        "0.1.0-test",
		JWTSecret:           "test-secret-key-with-at-least-32-characters",
		AllowedOrigins:      []string{"http://127.0.0.1:8080"},
		RateLimitUserPerMin: 5,
		RateLimitDevPerMin:  10,
		DrawMinIntervalMs:   400,
		WebDistDir:          webDir,
	}

	store := &stubStore{}
	authSvc := service.NewAuthService(cfg.JWTSecret)
	gachaSvc := service.NewGachaService(store, service.NewCryptoWeightedRNG(), cfg.AppEnv)
	sduiSvc := service.NewSDUIService(store)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	h := NewRouter(RouterDeps{
		Config:       cfg,
		Logger:       logger,
		AuthService:  authSvc,
		GachaService: gachaSvc,
		SDUIService:  sduiSvc,
		RateLimiter:  rl,
	})
	return h, authSvc
}

func TestSecurityHeadersAndCSPOnWebView(t *testing.T) {
	handler, _ := newTestRouter(t, NewRateLimiter(60, 90, 400*time.Millisecond))

	req := httptest.NewRequest(http.MethodGet, "/webview/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /webview/, got %d", rec.Code)
	}
	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'self'") || !strings.Contains(csp, "style-src 'self'") {
		t.Fatalf("unexpected CSP header: %q", csp)
	}
	if strings.Contains(csp, "unsafe-inline") || strings.Contains(csp, "unsafe-eval") {
		t.Fatalf("CSP must not allow unsafe-inline or unsafe-eval: %q", csp)
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("expected X-Frame-Options DENY, got %q", rec.Header().Get("X-Frame-Options"))
	}

	// Tanpa cookie admin, /swagger/ dan /sdui-admin/ harus redirect ke /login
	swaggerUnauthReq := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	swaggerUnauthRec := httptest.NewRecorder()
	handler.ServeHTTP(swaggerUnauthRec, swaggerUnauthReq)
	if swaggerUnauthRec.Code != http.StatusFound || !strings.HasPrefix(swaggerUnauthRec.Header().Get("Location"), "/login") {
		t.Fatalf("expected 302 redirect to /login on /swagger/, got %d", swaggerUnauthRec.Code)
	}

	adminUnauthReq := httptest.NewRequest(http.MethodGet, "/sdui-admin/", nil)
	adminUnauthRec := httptest.NewRecorder()
	handler.ServeHTTP(adminUnauthRec, adminUnauthReq)
	if adminUnauthRec.Code != http.StatusFound || !strings.HasPrefix(adminUnauthRec.Header().Get("Location"), "/login") {
		t.Fatalf("expected 302 redirect to /login on /sdui-admin/, got %d", adminUnauthRec.Code)
	}

	// Login admin terlebih dahulu
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("username=admin&password=LivinVillage2026%21"))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 on POST /login, got %d", loginRec.Code)
	}
	cookies := loginRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected admin session cookie after login")
	}

	swaggerReq := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	swaggerReq.AddCookie(cookies[0])
	swaggerRec := httptest.NewRecorder()
	handler.ServeHTTP(swaggerRec, swaggerReq)
	if swaggerRec.Code != http.StatusOK || !strings.Contains(swaggerRec.Body.String(), "/openapi.yaml") {
		t.Fatalf("expected Swagger UI HTML on /swagger/, got %d", swaggerRec.Code)
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/sdui-admin/", nil)
	adminReq.AddCookie(cookies[0])
	adminRec := httptest.NewRecorder()
	handler.ServeHTTP(adminRec, adminReq)
	if adminRec.Code != http.StatusOK || !strings.Contains(adminRec.Body.String(), "SDUI Configurator") {
		t.Fatalf("expected SDUI Admin HTML on /sdui-admin/, got %d", adminRec.Code)
	}
}

func TestTokenScopeIsolation(t *testing.T) {
	handler, authSvc := newTestRouter(t, NewRateLimiter(60, 90, 400*time.Millisecond))

	master, err := authSvc.IssueMasterSession("u1", "dev1")
	if err != nil {
		t.Fatalf("issue master: %v", err)
	}

	// Master token can mint short-lived webview token
	wvClaims := service.AuthClaims{UserID: "u1", DeviceID: "dev1", Scope: service.TokenScopeNative}
	wvTok, err := authSvc.IssueWebViewToken(wvClaims)
	if err != nil {
		t.Fatalf("issue webview token: %v", err)
	}
	if wvTok.TTLSeconds != 120 || wvTok.Scope != service.TokenScopeWebView {
		t.Fatalf("unexpected webview token metadata: %+v", wvTok)
	}

	// Webview token CANNOT mint another webview token or call synthetic topup
	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/v1/auth/webview-token", ""},
		{http.MethodPost, "/api/v1/gacha/synthetic-topup", `{"points":100}`},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Authorization", "Bearer "+wvTok.Token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s %s with webview token expected 403, got %d", tc.method, tc.path, rec.Code)
		}
	}

	// Master token works for topup
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gacha/synthetic-topup", strings.NewReader(`{"points":100}`))
	req.Header.Set("Authorization", "Bearer "+master.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("master token topup expected 200, got %d", rec.Code)
	}
}

func TestRateLimitAndAutomationBurstDetection(t *testing.T) {
	now := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)
	rl := NewRateLimiter(10, 15, 400*time.Millisecond)
	rl.nowFn = func() time.Time { return now }

	handler, authSvc := newTestRouter(t, rl)
	master, _ := authSvc.IssueMasterSession("u-bot", "dev-bot")

	doDraw := func(key string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/gacha/draw", strings.NewReader(`{"expected_table_version":"v1.0.0"}`))
		req.Header.Set("Authorization", "Bearer "+master.Token)
		req.Header.Set("X-Device-ID", "dev-bot")
		req.Header.Set("Idempotency-Key", key)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	// Draw 1: OK
	if code := doDraw("key-1"); code != http.StatusOK {
		t.Fatalf("draw 1 expected 200, got %d", code)
	}

	// Retry immediately with SAME key-1 (F6 retry) -> allowed!
	if code := doDraw("key-1"); code != http.StatusOK {
		t.Fatalf("same-key immediate retry expected 200, got %d", code)
	}

	// Draw with DIFFERENT key-2 within 100ms (< 400ms) -> blocked as automation!
	now = now.Add(100 * time.Millisecond)
	if code := doDraw("key-2"); code != http.StatusTooManyRequests {
		t.Fatalf("rapid distinct draw expected 429, got %d", code)
	}

	// After waiting 500ms -> allowed again
	now = now.Add(500 * time.Millisecond)
	if code := doDraw("key-3"); code != http.StatusOK {
		t.Fatalf("draw after spacing expected 200, got %d", code)
	}
}
