package domain

import "time"

// SDUIComponentType dibatasi pada katalog komponen gacha referensi (non-generik).
type SDUIComponentType string

const (
	SDUITypeText            SDUIComponentType = "text"
	SDUITypeButton          SDUIComponentType = "button"
	SDUITypeBalanceCard     SDUIComponentType = "balance_card"
	SDUITypeProbabilityList SDUIComponentType = "probability_list"
	SDUITypeGachaBox        SDUIComponentType = "gacha_box"
	SDUITypeHistoryList     SDUIComponentType = "history_list"

	// Komponen Village Quiz Full-Screen (isometric layout)
	SDUITypeFloatingBanner SDUIComponentType = "floating_banner"
	SDUITypeFloatingAction SDUIComponentType = "floating_action"
	SDUITypeMapPin         SDUIComponentType = "map_pin"
)

type SDUIActionType string

const (
	SDUIActionDraw          SDUIActionType = "draw"
	SDUIActionRefresh       SDUIActionType = "refresh"
	SDUIActionToggleHistory SDUIActionType = "toggle_history"

	// Aksi whitelist Village Quiz
	SDUIActionOpenQuiz    SDUIActionType = "open_quiz"
	SDUIActionOpenHelp    SDUIActionType = "open_help"
	SDUIActionOpenRewards SDUIActionType = "open_rewards"
)

type SDUINode struct {
	ID     string            `json:"id"`
	Type   SDUIComponentType `json:"type"`
	Props  map[string]any    `json:"props,omitempty"`
	Action *SDUIActionType   `json:"action,omitempty"`
}

type SDUIScreen struct {
	SchemaVersion    int              `json:"schema_version"`
	ScreenID         string           `json:"screen_id"`
	Variant          string           `json:"variant"`
	NavigationTitle  string           `json:"navigation_title,omitempty"`
	BackgroundAsset  string           `json:"background_asset,omitempty"`
	TableVersion     string           `json:"table_version"`
	SupportedActions []SDUIActionType `json:"supported_actions"`
	Components       []SDUINode       `json:"components"`
	GeneratedAt      time.Time        `json:"generated_at"`
}
