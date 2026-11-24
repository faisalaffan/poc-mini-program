package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
	"github.com/faisalaffan/poc-mini-program/backend/internal/service"
)

type VillageHandler struct {
	svc *service.GachaService
}

func NewVillageHandler(svc *service.GachaService) *VillageHandler {
	return &VillageHandler{svc: svc}
}

func (h *VillageHandler) GetState(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		mapDomainError(w, r, domain.ErrUnauthorized)
		return
	}
	state, err := h.svc.GetVillageState(r.Context(), claims.UserID)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

type submitQuizBody struct {
	SelectedIndex int `json:"selected_index"`
}

func (h *VillageHandler) SubmitQuiz(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		mapDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	quizID := strings.TrimSpace(r.PathValue("quizID"))
	if quizID == "" {
		mapDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	var body submitQuizBody
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		mapDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	deviceID := strings.TrimSpace(r.Header.Get("X-Device-ID"))
	if deviceID == "" {
		deviceID = claims.DeviceID
	}

	res, err := h.svc.SubmitQuiz(r.Context(), domain.QuizSubmitRequest{
		UserID:         claims.UserID,
		DeviceID:       deviceID,
		QuizID:         quizID,
		SelectedIndex:  body.SelectedIndex,
		IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		ClientApproach: domain.ParseClientApproach(strings.TrimSpace(r.Header.Get("X-Client-Approach"))),
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

type resetQuizzesBody struct {
	UserID string `json:"user_id"`
}

func (h *VillageHandler) ResetQuizzes(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		mapDomainError(w, r, domain.ErrUnauthorized)
		return
	}
	var body resetQuizzesBody
	if r.Body != nil {
		_ = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body)
	}
	userID := strings.TrimSpace(body.UserID)
	if userID == "" {
		userID = claims.UserID
	}
	deleted, err := h.svc.ResetUserQuizzes(r.Context(), userID)
	if err != nil {
		mapDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":             userID,
		"deleted_submissions": deleted,
	})
}
