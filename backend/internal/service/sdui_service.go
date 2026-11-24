package service

import (
	"context"
	"errors"
	"time"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
)

type SDUIService struct {
	store GachaStore
}

func NewSDUIService(store GachaStore) *SDUIService {
	return &SDUIService{store: store}
}

// BuildVillageScreen membaca layout SDUI dari database (sdui_screens) dan menghidrasi status selesai kuis per user.
func (s *SDUIService) BuildVillageScreen(ctx context.Context, userID, variant string, overrideSchemaVersion int) (domain.SDUIScreen, error) {
	return s.BuildVillageScreenWithOptions(ctx, userID, variant, overrideSchemaVersion, false)
}

func (s *SDUIService) BuildVillageScreenWithOptions(ctx context.Context, userID, variant string, overrideSchemaVersion int, includeAnswers bool) (domain.SDUIScreen, error) {
	if variant == "" {
		variant = "default"
	}

	screen, err := s.store.GetSDUIScreen(ctx, "livin_village_home", variant)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) && variant != "default" {
			screen, err = s.store.GetSDUIScreen(ctx, "livin_village_home", "default")
		}
		if err != nil {
			return domain.SDUIScreen{}, err
		}
	}

	if overrideSchemaVersion > 0 {
		screen.SchemaVersion = overrideSchemaVersion
	}

	state, err := s.store.GetVillageState(ctx, userID)
	if err != nil {
		return domain.SDUIScreen{}, err
	}
	table, err := s.store.GetActiveProbabilityTable(ctx)
	if err != nil {
		return domain.SDUIScreen{}, err
	}
	screen.TableVersion = table.TableVersion

	if screen.BackgroundAsset == "" || screen.BackgroundAsset == "bg-isometric" {
		screen.BackgroundAsset = "/assets/bg-isometric.png"
	}

	quizByID := make(map[string]domain.VillageQuiz, len(state.Quizzes))
	for _, q := range state.Quizzes {
		quizByID[q.ID] = q
	}

	for idx := range screen.Components {
		if screen.Components[idx].Type != domain.SDUITypeMapPin || screen.Components[idx].Props == nil {
			continue
		}
		props := screen.Components[idx].Props
		quizID, ok := props["quiz_id"].(string)
		if !ok {
			continue
		}
		q, found := quizByID[quizID]
		if !found {
			props["completed"] = false
			continue
		}
		props["completed"] = q.Completed
		if v, ok := props["building_name"].(string); !ok || v == "" {
			props["building_name"] = q.BuildingName
		}
		if v, ok := props["title"].(string); !ok || v == "" {
			props["title"] = q.Title
		}
		if _, ok := props["reward_points"]; !ok {
			props["reward_points"] = q.RewardPoints
		}
		if v, ok := props["question"].(string); !ok || v == "" {
			props["question"] = q.Question
		}
		if _, ok := props["options"]; !ok {
			props["options"] = q.Options
		}
		if includeAnswers {
			if _, ok := props["correct_index"]; !ok {
				props["correct_index"] = q.CorrectIndex
			}
		} else {
			delete(props, "correct_index")
		}
	}

	return screen, nil
}

func (s *SDUIService) UpdateVillageScreen(ctx context.Context, screen domain.SDUIScreen) (domain.SDUIScreen, error) {
	if screen.ScreenID == "" {
		screen.ScreenID = "livin_village_home"
	}
	if screen.Variant == "" {
		screen.Variant = "default"
	}
	if screen.SchemaVersion <= 0 {
		screen.SchemaVersion = 1
	}
	if screen.NavigationTitle == "" {
		screen.NavigationTitle = "Livin' Village"
	}
	if screen.BackgroundAsset == "" {
		screen.BackgroundAsset = "/assets/bg-isometric.png"
	}
	if len(screen.Components) == 0 {
		return domain.SDUIScreen{}, domain.ErrInvalidInput
	}
	return s.store.UpsertSDUIScreen(ctx, screen)
}

// BuildGachaScreen tetap dipertahankan untuk kompatibilitas endpoint /api/v1/sdui/gacha-screen.
func (s *SDUIService) BuildGachaScreen(ctx context.Context, variant string, overrideSchemaVersion int) (domain.SDUIScreen, error) {
	table, err := s.store.GetActiveProbabilityTable(ctx)
	if err != nil {
		return domain.SDUIScreen{}, err
	}

	schemaVersion := 1
	if overrideSchemaVersion > 0 {
		schemaVersion = overrideSchemaVersion
	}
	if variant == "" {
		variant = "default"
	}

	drawAction := domain.SDUIActionDraw
	refreshAction := domain.SDUIActionRefresh
	nodes := []domain.SDUINode{
		{
			ID:   "balance_summary",
			Type: domain.SDUITypeBalanceCard,
			Props: map[string]any{
				"title": "Saldo Poin Terkini",
			},
		},
		{
			ID:     "gacha_box_main",
			Type:   domain.SDUITypeGachaBox,
			Props:  map[string]any{"title": "Kotak Hadiah", "cta_label": "Buka Kotak (100 Poin)"},
			Action: &drawAction,
		},
		{
			ID:     "refresh_btn",
			Type:   domain.SDUITypeButton,
			Props:  map[string]any{"label": "Segarkan"},
			Action: &refreshAction,
		},
	}

	return domain.SDUIScreen{
		SchemaVersion:    schemaVersion,
		ScreenID:         "gacha_home",
		Variant:          variant,
		TableVersion:     table.TableVersion,
		SupportedActions: []domain.SDUIActionType{drawAction, refreshAction, domain.SDUIActionToggleHistory},
		Components:       nodes,
		GeneratedAt:      time.Now().UTC(),
	}, nil
}
