package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/faisalaffan/poc-mini-program/backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

func (r *GachaRepository) GetSDUIScreen(ctx context.Context, screenID, variant string) (domain.SDUIScreen, error) {
	var screen domain.SDUIScreen
	var actionsRaw []byte
	var componentsRaw []byte
	var updatedAt time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT schema_version, screen_id, variant, navigation_title, background_asset, supported_actions, components, updated_at
		FROM sdui_screens
		WHERE screen_id = $1 AND variant = $2
	`, screenID, variant).Scan(
		&screen.SchemaVersion,
		&screen.ScreenID,
		&screen.Variant,
		&screen.NavigationTitle,
		&screen.BackgroundAsset,
		&actionsRaw,
		&componentsRaw,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.SDUIScreen{}, domain.ErrNotFound
		}
		return domain.SDUIScreen{}, fmt.Errorf("query sdui_screens: %w", err)
	}

	if err := json.Unmarshal(actionsRaw, &screen.SupportedActions); err != nil {
		return domain.SDUIScreen{}, fmt.Errorf("unmarshal supported_actions: %w", err)
	}
	if err := json.Unmarshal(componentsRaw, &screen.Components); err != nil {
		return domain.SDUIScreen{}, fmt.Errorf("unmarshal components: %w", err)
	}
	screen.GeneratedAt = updatedAt.UTC()
	return screen, nil
}

func (r *GachaRepository) UpsertSDUIScreen(ctx context.Context, screen domain.SDUIScreen) (domain.SDUIScreen, error) {
	for idx := range screen.Components {
		if screen.Components[idx].Type != domain.SDUITypeMapPin {
			continue
		}
		if screen.Components[idx].Props == nil {
			screen.Components[idx].Props = map[string]any{}
		}
		props := screen.Components[idx].Props
		if qid, _ := props["quiz_id"].(string); qid == "" {
			props["quiz_id"] = screen.Components[idx].ID
		}
	}

	actionsRaw, err := json.Marshal(screen.SupportedActions)
	if err != nil {
		return domain.SDUIScreen{}, fmt.Errorf("marshal supported_actions: %w", err)
	}
	componentsRaw, err := json.Marshal(screen.Components)
	if err != nil {
		return domain.SDUIScreen{}, fmt.Errorf("marshal components: %w", err)
	}

	var updatedAt time.Time
	err = r.pool.QueryRow(ctx, `
		INSERT INTO sdui_screens (
			screen_id, variant, schema_version, navigation_title, background_asset,
			supported_actions, components, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, NOW())
		ON CONFLICT (screen_id, variant) DO UPDATE SET
			schema_version = EXCLUDED.schema_version,
			navigation_title = EXCLUDED.navigation_title,
			background_asset = EXCLUDED.background_asset,
			supported_actions = EXCLUDED.supported_actions,
			components = EXCLUDED.components,
			updated_at = NOW()
		RETURNING updated_at
	`,
		screen.ScreenID,
		screen.Variant,
		screen.SchemaVersion,
		screen.NavigationTitle,
		screen.BackgroundAsset,
		string(actionsRaw),
		string(componentsRaw),
	).Scan(&updatedAt)
	if err != nil {
		return domain.SDUIScreen{}, fmt.Errorf("upsert sdui_screens: %w", err)
	}

	if err := r.syncQuizzesFromComponents(ctx, screen.Components); err != nil {
		return domain.SDUIScreen{}, err
	}

	screen.GeneratedAt = updatedAt.UTC()
	return screen, nil
}

func (r *GachaRepository) syncQuizzesFromComponents(ctx context.Context, components []domain.SDUINode) error {
	for idx, node := range components {
		if node.Type != domain.SDUITypeMapPin || node.Props == nil {
			continue
		}
		quizID, _ := node.Props["quiz_id"].(string)
		if quizID == "" {
			quizID = node.ID
		}
		question, _ := node.Props["question"].(string)
		options := extractStringSlice(node.Props["options"])
		if quizID == "" || question == "" || len(options) == 0 {
			continue
		}

		title, _ := node.Props["title"].(string)
		if title == "" {
			title = "Kuis"
		}
		buildingName, _ := node.Props["building_name"].(string)
		if buildingName == "" {
			buildingName = title
		}
		pinColor, _ := node.Props["pin_color"].(string)
		if pinColor != "blue" && pinColor != "orange" && pinColor != "red" {
			pinColor = "blue"
		}
		xRatio := extractFloat64(node.Props["x_ratio"], 0.5)
		yRatio := extractFloat64(node.Props["y_ratio"], 0.5)
		rewardPoints := extractInt64(node.Props["reward_points"], 150)
		correctIndex := int(extractInt64(node.Props["correct_index"], 0))
		if correctIndex < 0 || correctIndex >= len(options) {
			correctIndex = 0
		}
		optionsRaw, err := json.Marshal(options)
		if err != nil {
			return fmt.Errorf("marshal quiz options: %w", err)
		}

		if _, err := r.pool.Exec(ctx, `
			INSERT INTO village_quizzes (
				id, building_name, title, pin_color, x_ratio, y_ratio,
				reward_points, question, options, correct_index, explanation, display_order
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11, $12)
			ON CONFLICT (id) DO UPDATE SET
				building_name = EXCLUDED.building_name,
				title = EXCLUDED.title,
				pin_color = EXCLUDED.pin_color,
				x_ratio = EXCLUDED.x_ratio,
				y_ratio = EXCLUDED.y_ratio,
				reward_points = EXCLUDED.reward_points,
				question = EXCLUDED.question,
				options = EXCLUDED.options,
				correct_index = EXCLUDED.correct_index
		`,
			quizID, buildingName, title, pinColor, xRatio, yRatio,
			rewardPoints, question, string(optionsRaw), correctIndex,
			"Jawaban Anda benar! Poin hadiah telah masuk ke ledger.", idx+1,
		); err != nil {
			return fmt.Errorf("sync quiz %s: %w", quizID, err)
		}
	}
	return nil
}

func extractStringSlice(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func extractFloat64(raw any, fallback float64) float64 {
	switch v := raw.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return fallback
	}
}

func extractInt64(raw any, fallback int64) int64 {
	switch v := raw.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return fallback
	}
}
