package http

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
	"github.com/faisalaffan/poc-mini-program/backend/internal/service"
	"github.com/google/uuid"
)

type ctxKey string

const (
	ctxKeyRequestID ctxKey = "request_id"
	ctxKeyClaims    ctxKey = "auth_claims"
)

func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyRequestID).(string); ok {
		return v
	}
	return ""
}

func ClaimsFromContext(ctx context.Context) (service.AuthClaims, bool) {
	v, ok := ctx.Value(ctxKeyClaims).(service.AuthClaims)
	return v, ok
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func RequestLoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
			if reqID == "" {
				reqID = "req_" + uuid.NewString()
			}
			w.Header().Set("X-Request-ID", reqID)
			ctx := context.WithValue(r.Context(), ctxKeyRequestID, reqID)

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(rec, r.WithContext(ctx))

			logger.Info("http_request",
				"request_id", reqID,
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"approach", r.Header.Get("X-Client-Approach"),
			)
		})
	}
}

func SecurityHeadersMiddleware(buildVersion string, allowedOrigins []string) func(http.Handler) http.Handler {
	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
			w.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
			w.Header().Set("X-App-Build-Version", buildVersion)

			origin := r.Header.Get("Origin")
			if _, ok := originSet[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Device-ID, X-Client-Approach, X-Request-ID")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
			}

			if strings.HasPrefix(r.URL.Path, "/webview") {
				w.Header().Set("Content-Security-Policy",
					"default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; font-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
				w.Header().Set("Cache-Control", "no-store")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func AuthMiddleware(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Header Authorization Bearer wajib disertakan.")
				return
			}
			rawToken := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
			claims, err := auth.VerifyToken(rawToken)
			if err != nil {
				mapDomainError(w, r, err)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKeyClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type slidingWindow struct {
	timestamps []time.Time
}

type drawPaceRecord struct {
	lastAt  time.Time
	lastKey string
}

// RateLimiter mengelola batas permintaan per user, per device, serta deteksi burst automasi draw.
type RateLimiter struct {
	mu             sync.Mutex
	userLimit      int
	deviceLimit    int
	minDrawSpacing time.Duration
	window         time.Duration
	nowFn          func() time.Time
	userHits       map[string]*slidingWindow
	deviceHits     map[string]*slidingWindow
	lastDraws      map[string]drawPaceRecord
}

func NewRateLimiter(userPerMin, devicePerMin int, minDrawInterval time.Duration) *RateLimiter {
	return &RateLimiter{
		userLimit:      userPerMin,
		deviceLimit:    devicePerMin,
		minDrawSpacing: minDrawInterval,
		window:         time.Minute,
		nowFn:          time.Now,
		userHits:       make(map[string]*slidingWindow),
		deviceHits:     make(map[string]*slidingWindow),
		lastDraws:      make(map[string]drawPaceRecord),
	}
}

func (rl *RateLimiter) AllowRequest(userID, deviceID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.nowFn()
	if !rl.checkAndRecord(rl.userHits, userID, rl.userLimit, now) {
		return false
	}
	if deviceID != "" && !rl.checkAndRecord(rl.deviceHits, deviceID, rl.deviceLimit, now) {
		return false
	}
	return true
}

// CheckDrawPace mendeteksi burst rapid draw untuk idempotency key berbeda dalam jeda terlalu singkat.
// Retry dengan idempotency key yang sama tetap diizinkan agar F6 retry tidak terblokir.
func (rl *RateLimiter) CheckDrawPace(userID, idempotencyKey string) bool {
	if rl.minDrawSpacing <= 0 {
		return true
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.nowFn()
	prev, exists := rl.lastDraws[userID]
	if exists && prev.lastKey != idempotencyKey && now.Sub(prev.lastAt) < rl.minDrawSpacing {
		return false
	}
	rl.lastDraws[userID] = drawPaceRecord{lastAt: now, lastKey: idempotencyKey}
	return true
}

func (rl *RateLimiter) checkAndRecord(store map[string]*slidingWindow, key string, limit int, now time.Time) bool {
	if limit <= 0 || key == "" {
		return true
	}
	sw, ok := store[key]
	if !ok {
		sw = &slidingWindow{}
		store[key] = sw
	}
	cutoff := now.Add(-rl.window)
	kept := sw.timestamps[:0]
	for _, ts := range sw.timestamps {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= limit {
		sw.timestamps = kept
		return false
	}
	sw.timestamps = append(kept, now)
	return true
}

func RateLimitMiddleware(rl *RateLimiter, auditSink func(context.Context, domain.AuditEvent)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			deviceID := strings.TrimSpace(r.Header.Get("X-Device-ID"))
			if deviceID == "" {
				deviceID = claims.DeviceID
			}

			if !rl.AllowRequest(claims.UserID, deviceID) {
				if auditSink != nil {
					auditSink(r.Context(), domain.AuditEvent{
						UserID:         claims.UserID,
						DeviceID:       deviceID,
						Action:         r.Method + " " + r.URL.Path,
						Outcome:        "REJECTED_RATE_LIMIT",
						ClientApproach: r.Header.Get("X-Client-Approach"),
						AutomationFlag: true,
					})
				}
				w.Header().Set("Retry-After", "5")
				mapDomainError(w, r, domain.ErrRateLimited)
				return
			}

			if r.Method == http.MethodPost && r.URL.Path == "/api/v1/gacha/draw" {
				idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
				if idemKey != "" && !rl.CheckDrawPace(claims.UserID, idemKey) {
					if auditSink != nil {
						auditSink(r.Context(), domain.AuditEvent{
							UserID:         claims.UserID,
							DeviceID:       deviceID,
							Action:         "DRAW",
							Outcome:        "REJECTED_AUTOMATION_BURST",
							IdempotencyKey: idemKey,
							ClientApproach: r.Header.Get("X-Client-Approach"),
							AutomationFlag: true,
						})
					}
					w.Header().Set("Retry-After", "1")
					mapDomainError(w, r, domain.ErrAutomationDetected)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
