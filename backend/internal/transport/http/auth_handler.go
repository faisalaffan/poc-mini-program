package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
	"github.com/faisalaffan/poc-mini-program/backend/internal/service"
)

type AuthHandler struct {
	auth  *service.AuthService
	gacha *service.GachaService
}

func NewAuthHandler(auth *service.AuthService, gacha *service.GachaService) *AuthHandler {
	return &AuthHandler{auth: auth, gacha: gacha}
}

type sessionRequest struct {
	UserID   string `json:"user_id"`
	DeviceID string `json:"device_id"`
}

func (h *AuthHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req sessionRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		userID = "demo-user-001"
	}
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		deviceID = strings.TrimSpace(r.Header.Get("X-Device-ID"))
	}
	if deviceID == "" {
		deviceID = "ios-poc-device"
	}

	if err := h.gacha.EnsureUser(r.Context(), userID); err != nil {
		mapDomainError(w, r, err)
		return
	}

	issued, err := h.auth.IssueMasterSession(userID, deviceID)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, issued)
}

func (h *AuthHandler) IssueWebViewToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		mapDomainError(w, r, domain.ErrUnauthorized)
		return
	}
	issued, err := h.auth.IssueWebViewToken(claims)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, issued)
}
