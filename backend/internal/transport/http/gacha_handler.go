package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
	"github.com/faisalaffan/poc-mini-program/backend/internal/service"
)

type GachaHandler struct {
	svc *service.GachaService
}

func NewGachaHandler(svc *service.GachaService) *GachaHandler {
	return &GachaHandler{svc: svc}
}

func (h *GachaHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *GachaHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.HealthPing(r.Context()); err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "DB_UNAVAILABLE", "PostgreSQL belum siap.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *GachaHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		mapDomainError(w, r, domain.ErrUnauthorized)
		return
	}
	bal, err := h.svc.GetBalance(r.Context(), claims.UserID)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, bal)
}

func (h *GachaHandler) GetProbabilities(w http.ResponseWriter, r *http.Request) {
	table, err := h.svc.GetProbabilities(r.Context())
	if err != nil {
		mapDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, table)
}

type drawRequestBody struct {
	ExpectedTableVersion string `json:"expected_table_version"`
}

func (h *GachaHandler) ExecuteDraw(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		mapDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	var body drawRequestBody
	if r.Body != nil {
		decErr := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body)
		if decErr != nil && decErr != io.EOF {
			mapDomainError(w, r, domain.ErrInvalidInput)
			return
		}
	}

	deviceID := strings.TrimSpace(r.Header.Get("X-Device-ID"))
	if deviceID == "" {
		deviceID = claims.DeviceID
	}

	res, err := h.svc.ExecuteDraw(r.Context(), domain.DrawRequest{
		UserID:               claims.UserID,
		DeviceID:             deviceID,
		IdempotencyKey:       strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		ExpectedTableVersion: strings.TrimSpace(body.ExpectedTableVersion),
		ClientApproach:       domain.ParseClientApproach(strings.TrimSpace(r.Header.Get("X-Client-Approach"))),
	})
	if err != nil {
		mapDomainError(w, r, err)
		return
	}

	if res.IdempotencyReplayed {
		w.Header().Set("X-Idempotency-Replayed", "true")
	} else {
		w.Header().Set("X-Idempotency-Replayed", "false")
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *GachaHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		mapDomainError(w, r, domain.ErrUnauthorized)
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	items, err := h.svc.GetHistory(r.Context(), claims.UserID, limit)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type topUpBody struct {
	Points int64 `json:"points"`
}

func (h *GachaHandler) SyntheticTopUp(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		mapDomainError(w, r, domain.ErrUnauthorized)
		return
	}
	if claims.Scope != service.TokenScopeNative {
		mapDomainError(w, r, domain.ErrForbiddenScope)
		return
	}

	var body topUpBody
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		mapDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	bal, err := h.svc.SyntheticTopUp(r.Context(), claims.UserID, claims.DeviceID, body.Points)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, bal)
}

func (h *GachaHandler) Reconcile(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		mapDomainError(w, r, domain.ErrUnauthorized)
		return
	}
	rep, err := h.svc.Reconcile(r.Context(), claims.UserID)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}
