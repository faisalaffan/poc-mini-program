package domain

import "time"

// ClientApproach merepresentasikan sumber front-end POC sesuai PRD.
type ClientApproach string

const (
	ApproachWebView ClientApproach = "webview"
	ApproachNative  ClientApproach = "native"
	ApproachSDUI    ClientApproach = "sdui"
)

func ParseClientApproach(raw string) ClientApproach {
	switch ClientApproach(raw) {
	case ApproachWebView, ApproachNative, ApproachSDUI:
		return ClientApproach(raw)
	default:
		return ApproachNative
	}
}

type Rarity string

const (
	RarityCommon   Rarity = "COMMON"
	RarityUncommon Rarity = "UNCOMMON"
	RarityRare     Rarity = "RARE"
	RarityEpic     Rarity = "EPIC"
	RarityJackpot  Rarity = "JACKPOT"
)

type RewardTier struct {
	ID                 string  `json:"id"`
	TableVersion       string  `json:"-"`
	Code               string  `json:"code"`
	Name               string  `json:"name"`
	Rarity             Rarity  `json:"rarity"`
	RewardPoints       int64   `json:"reward_points"`
	WeightBPS          int     `json:"weight_bps"`
	ProbabilityPercent float64 `json:"probability_percent"`
	DisplayOrder       int     `json:"display_order"`
}

type ProbabilityTable struct {
	TableVersion   string       `json:"table_version"`
	DrawCost       int64        `json:"draw_cost"`
	TotalWeightBPS int          `json:"total_weight_bps"`
	Tiers          []RewardTier `json:"tiers"`
}

type DrawRequest struct {
	UserID               string
	DeviceID             string
	IdempotencyKey       string
	ExpectedTableVersion string
	ClientApproach       ClientApproach
	RequestHash          string
}

type DrawResult struct {
	DrawID              string         `json:"draw_id"`
	IdempotencyKey      string         `json:"idempotency_key"`
	IdempotencyReplayed bool           `json:"idempotency_replayed"`
	TableVersion        string         `json:"table_version"`
	DrawCost            int64          `json:"draw_cost"`
	Reward              RewardTier     `json:"reward"`
	BalanceBefore       int64          `json:"balance_before"`
	BalanceAfter        int64          `json:"balance_after"`
	ClientApproach      ClientApproach `json:"client_approach"`
	CreatedAt           time.Time      `json:"created_at"`
}

type HistoryItem struct {
	DrawID         string         `json:"draw_id"`
	TableVersion   string         `json:"table_version"`
	DrawCost       int64          `json:"draw_cost"`
	RewardCode     string         `json:"reward_code"`
	RewardName     string         `json:"reward_name"`
	Rarity         Rarity         `json:"rarity"`
	RewardPoints   int64          `json:"reward_points"`
	BalanceAfter   int64          `json:"balance_after"`
	ClientApproach ClientApproach `json:"client_approach"`
	CreatedAt      time.Time      `json:"created_at"`
}
