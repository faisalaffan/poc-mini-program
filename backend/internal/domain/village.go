package domain

import "time"

type PinColor string

const (
	PinColorBlue   PinColor = "blue"
	PinColorOrange PinColor = "orange"
	PinColorRed    PinColor = "red"
)

type VillageQuiz struct {
	ID           string   `json:"id"`
	BuildingName string   `json:"building_name"`
	Title        string   `json:"title"`
	PinColor     PinColor `json:"pin_color"`
	XRatio       float64  `json:"x_ratio"`
	YRatio       float64  `json:"y_ratio"`
	RewardPoints int64    `json:"reward_points"`
	Question     string   `json:"question"`
	Options      []string `json:"options"`
	CorrectIndex int      `json:"-"`
	Completed    bool     `json:"completed"`
	DisplayOrder int      `json:"display_order"`
}

type VillageState struct {
	UserID         string        `json:"user_id"`
	CampaignBanner string        `json:"campaign_banner"`
	Balance        int64         `json:"balance"`
	CompletedCount int           `json:"completed_count"`
	TotalQuizzes   int           `json:"total_quizzes"`
	Quizzes        []VillageQuiz `json:"quizzes"`
	Rules          []string      `json:"rules"`
}

type QuizSubmitRequest struct {
	UserID         string
	DeviceID       string
	QuizID         string
	SelectedIndex  int
	IdempotencyKey string
	ClientApproach ClientApproach
	RequestHash    string
}

type QuizSubmitResult struct {
	SubmissionID        string         `json:"submission_id"`
	QuizID              string         `json:"quiz_id"`
	IdempotencyKey      string         `json:"idempotency_key"`
	IdempotencyReplayed bool           `json:"idempotency_replayed"`
	Correct             bool           `json:"correct"`
	RewardPoints        int64          `json:"reward_points"`
	BalanceBefore       int64          `json:"balance_before"`
	BalanceAfter        int64          `json:"balance_after"`
	Explanation         string         `json:"explanation"`
	ClientApproach      ClientApproach `json:"client_approach"`
	CreatedAt           time.Time      `json:"created_at"`
}
