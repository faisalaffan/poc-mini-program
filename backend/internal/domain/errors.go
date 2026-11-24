package domain

import "errors"

var (
	ErrUnauthorized               = errors.New("unauthorized")
	ErrForbiddenScope             = errors.New("token scope forbidden for this operation")
	ErrInvalidInput               = errors.New("invalid input")
	ErrMissingIdempotencyKey      = errors.New("missing Idempotency-Key header")
	ErrMissingDeviceID            = errors.New("missing X-Device-ID header")
	ErrIdempotencyPayloadMismatch = errors.New("idempotency key reused with different request payload")
	ErrTableVersionMismatch       = errors.New("probability table version mismatch")
	ErrInsufficientBalance        = errors.New("insufficient point balance")
	ErrRateLimited                = errors.New("rate limit exceeded")
	ErrAutomationDetected         = errors.New("suspected automated rapid requests")
	ErrNotFound                   = errors.New("resource not found")
	ErrPOCOnly                    = errors.New("endpoint only available in poc environment")
	ErrQuizAlreadyCompleted       = errors.New("quiz already completed")
)
