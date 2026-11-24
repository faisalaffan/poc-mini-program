package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
)

type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	reqID := RequestIDFromContext(r.Context())
	writeJSON(w, status, errorEnvelope{
		Error: errorBody{
			Code:      code,
			Message:   message,
			RequestID: reqID,
		},
	})
}

func mapDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak valid atau sudah kadaluarsa.")
	case errors.Is(err, domain.ErrForbiddenScope):
		writeError(w, r, http.StatusForbidden, "FORBIDDEN_SCOPE", "Token tidak memiliki izin untuk aksi ini.")
	case errors.Is(err, domain.ErrMissingIdempotencyKey):
		writeError(w, r, http.StatusBadRequest, "MISSING_IDEMPOTENCY_KEY", "Header Idempotency-Key wajib diisi.")
	case errors.Is(err, domain.ErrMissingDeviceID):
		writeError(w, r, http.StatusBadRequest, "MISSING_DEVICE_ID", "Header X-Device-ID wajib diisi.")
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, r, http.StatusBadRequest, "INVALID_INPUT", "Parameter permintaan tidak valid.")
	case errors.Is(err, domain.ErrIdempotencyPayloadMismatch):
		writeError(w, r, http.StatusConflict, "IDEMPOTENCY_PAYLOAD_MISMATCH", "Idempotency-Key sudah dipakai dengan parameter berbeda.")
	case errors.Is(err, domain.ErrTableVersionMismatch):
		writeError(w, r, http.StatusConflict, "TABLE_VERSION_MISMATCH", "Versi tabel probabilitas berubah. Muat ulang probabilitas sebelum draw.")
	case errors.Is(err, domain.ErrInsufficientBalance):
		writeError(w, r, http.StatusUnprocessableEntity, "INSUFFICIENT_BALANCE", "Saldo poin tidak mencukupi untuk melakukan draw.")
	case errors.Is(err, domain.ErrRateLimited):
		writeError(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "Terlalu banyak permintaan. Coba lagi sebentar lagi.")
	case errors.Is(err, domain.ErrAutomationDetected):
		writeError(w, r, http.StatusTooManyRequests, "AUTOMATION_DETECTED", "Permintaan draw terlalu cepat dan terdeteksi sebagai automasi.")
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "NOT_FOUND", "Data tidak ditemukan.")
	case errors.Is(err, domain.ErrPOCOnly):
		writeError(w, r, http.StatusForbidden, "POC_ONLY", "Endpoint ini hanya tersedia di lingkungan POC.")
	case errors.Is(err, domain.ErrQuizAlreadyCompleted):
		writeError(w, r, http.StatusConflict, "QUIZ_ALREADY_COMPLETED", "Kuis di gedung ini sudah selesai dan hadiah poin sudah diklaim.")
	default:
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan internal server.")
	}
}
