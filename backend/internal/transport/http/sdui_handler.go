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

type SDUIHandler struct {
	svc *service.SDUIService
}

func NewSDUIHandler(svc *service.SDUIService) *SDUIHandler {
	return &SDUIHandler{svc: svc}
}

func (h *SDUIHandler) GetVillageScreen(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		mapDomainError(w, r, domain.ErrUnauthorized)
		return
	}
	variant := strings.TrimSpace(r.URL.Query().Get("variant"))
	overrideVersion := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("simulate_schema_version")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			overrideVersion = parsed
		}
	}

	includeAnswers := r.URL.Query().Get("include_answers") == "true"
	screen, err := h.svc.BuildVillageScreenWithOptions(r.Context(), claims.UserID, variant, overrideVersion, includeAnswers)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, screen)
}

func (h *SDUIHandler) UpdateVillageScreen(w http.ResponseWriter, r *http.Request) {
	var body domain.SDUIScreen
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<18)).Decode(&body); err != nil {
		mapDomainError(w, r, domain.ErrInvalidInput)
		return
	}
	updated, err := h.svc.UpdateVillageScreen(r.Context(), body)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *SDUIHandler) GetGachaScreen(w http.ResponseWriter, r *http.Request) {
	variant := strings.TrimSpace(r.URL.Query().Get("variant"))
	overrideVersion := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("simulate_schema_version")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			overrideVersion = parsed
		}
	}

	screen, err := h.svc.BuildGachaScreen(r.Context(), variant, overrideVersion)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, screen)
}
