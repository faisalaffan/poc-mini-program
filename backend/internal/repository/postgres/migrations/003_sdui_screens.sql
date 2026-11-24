CREATE TABLE IF NOT EXISTS sdui_screens (
    screen_id TEXT NOT NULL,
    variant TEXT NOT NULL,
    schema_version INT NOT NULL DEFAULT 1,
    navigation_title TEXT NOT NULL,
    background_asset TEXT NOT NULL,
    supported_actions JSONB NOT NULL,
    components JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (screen_id, variant)
);
