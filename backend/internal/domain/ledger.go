package domain

import "time"

type Balance struct {
	UserID       string    `json:"user_id"`
	Balance      int64     `json:"balance"`
	LastLedgerID int64     `json:"last_ledger_id"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ReconcileReport struct {
	UserID            string `json:"user_id"`
	LedgerBalance     int64  `json:"ledger_balance"`
	ComputedBalance   int64  `json:"computed_balance"`
	TotalSeeded       int64  `json:"total_seeded"`
	TotalQuizRewards  int64  `json:"total_quiz_rewards"`
	TotalDrawCost     int64  `json:"total_draw_cost"`
	TotalRewardPoints int64  `json:"total_reward_points"`
	TotalDraws        int    `json:"total_draws"`
	Balanced          bool   `json:"balanced"`
}

type AuditEvent struct {
	UserID         string
	DeviceID       string
	Action         string
	Outcome        string
	TableVersion   string
	DrawID         *string
	IdempotencyKey string
	ClientApproach string
	AutomationFlag bool
	Metadata       map[string]any
}
