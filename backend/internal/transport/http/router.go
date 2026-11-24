package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/faisalaffan/poc-mini-program/backend/internal/config"
	"github.com/faisalaffan/poc-mini-program/backend/internal/service"
)

type RouterDeps struct {
	Config       config.Config
	Logger       *slog.Logger
	AuthService  *service.AuthService
	GachaService *service.GachaService
	SDUIService  *service.SDUIService
	RateLimiter  *RateLimiter
}

func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	authH := NewAuthHandler(deps.AuthService, deps.GachaService)
	gachaH := NewGachaHandler(deps.GachaService)
	villageH := NewVillageHandler(deps.GachaService)
	sduiH := NewSDUIHandler(deps.SDUIService)
	spaH := NewSPAHandler(deps.Config.WebDistDir)

	if deps.RateLimiter == nil {
		deps.RateLimiter = NewRateLimiter(
			deps.Config.RateLimitUserPerMin,
			deps.Config.RateLimitDevPerMin,
			time.Duration(deps.Config.DrawMinIntervalMs)*time.Millisecond,
		)
	}

	authMW := AuthMiddleware(deps.AuthService)
	rlMW := RateLimitMiddleware(deps.RateLimiter, deps.GachaService.RecordSecurityAudit)
	protect := func(h http.HandlerFunc) http.Handler {
		return authMW(rlMW(h))
	}

	// Probes & Auth
	mux.HandleFunc("GET /healthz", gachaH.Healthz)
	mux.HandleFunc("GET /readyz", gachaH.Readyz)
	mux.HandleFunc("POST /api/v1/auth/session", authH.CreateSession)
	mux.Handle("POST /api/v1/auth/webview-token", protect(authH.IssueWebViewToken))

	// Gacha Endpoints (F1 - F6)
	mux.Handle("GET /api/v1/gacha/balance", protect(gachaH.GetBalance))
	mux.Handle("GET /api/v1/gacha/probabilities", protect(gachaH.GetProbabilities))
	mux.Handle("POST /api/v1/gacha/draw", protect(gachaH.ExecuteDraw))
	mux.Handle("GET /api/v1/gacha/history", protect(gachaH.GetHistory))
	mux.Handle("POST /api/v1/gacha/synthetic-topup", protect(gachaH.SyntheticTopUp))
	mux.Handle("GET /api/v1/gacha/reconcile", protect(gachaH.Reconcile))

	// Village Quiz Endpoints (Livin' Village)
	mux.Handle("GET /api/v1/village/state", protect(villageH.GetState))
	mux.Handle("POST /api/v1/village/quizzes/{quizID}/submit", protect(villageH.SubmitQuiz))
	mux.Handle("POST /api/v1/village/quizzes/reset", protect(villageH.ResetQuizzes))

	// SDUI (Pendekatan C)
	mux.Handle("GET /api/v1/sdui/village-screen", protect(sduiH.GetVillageScreen))
	mux.Handle("PUT /api/v1/sdui/village-screen", protect(sduiH.UpdateVillageScreen))
	mux.Handle("GET /api/v1/sdui/gacha-screen", protect(sduiH.GetGachaScreen))

	// Swagger UI & OpenAPI Spec
	mux.HandleFunc("GET /openapi.yaml", NewOpenAPIHandler(deps.Config.OpenAPIPath))
	mux.Handle("GET /swagger", http.RedirectHandler("/swagger/", http.StatusMovedPermanently))
	mux.HandleFunc("GET /swagger/", NewSwaggerUIHandler())

	// SDUI Visual Configurator (Admin UI) & Static Map Assets
	mux.Handle("GET /sdui-admin", http.RedirectHandler("/sdui-admin/", http.StatusMovedPermanently))
	mux.HandleFunc("GET /sdui-admin/", NewSDUIAdminHandler())
	bgHandler := NewSDUIBackgroundHandler(deps.Config.WebDistDir)
	mux.HandleFunc("GET /sdui-admin/bg-isometric.png", bgHandler)
	mux.HandleFunc("GET /assets/bg-isometric.png", bgHandler)

	// Vue 3 SPA Static Assets (Pendekatan A)
	mux.Handle("GET /webview", http.RedirectHandler("/webview/", http.StatusMovedPermanently))
	mux.Handle("GET /webview/", spaH)

	handler := SecurityHeadersMiddleware(deps.Config.BuildVersion, deps.Config.AllowedOrigins)(mux)
	handler = RequestLoggingMiddleware(deps.Logger)(handler)
	return handler
}
