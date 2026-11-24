CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS probability_tables (
    version TEXT PRIMARY KEY,
    draw_cost BIGINT NOT NULL CHECK (draw_cost > 0),
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_probability_tables_single_active
    ON probability_tables (is_active) WHERE is_active = TRUE;

CREATE TABLE IF NOT EXISTS reward_tiers (
    id TEXT PRIMARY KEY,
    table_version TEXT NOT NULL REFERENCES probability_tables(version),
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    rarity TEXT NOT NULL CHECK (rarity IN ('COMMON', 'UNCOMMON', 'RARE', 'EPIC', 'JACKPOT')),
    reward_points BIGINT NOT NULL CHECK (reward_points >= 0),
    weight_bps INT NOT NULL CHECK (weight_bps > 0 AND weight_bps <= 10000),
    display_order INT NOT NULL
);

CREATE TABLE IF NOT EXISTS gacha_draws (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id),
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    table_version TEXT NOT NULL REFERENCES probability_tables(version),
    reward_tier_id TEXT NOT NULL REFERENCES reward_tiers(id),
    draw_cost BIGINT NOT NULL,
    reward_points BIGINT NOT NULL,
    balance_before BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    client_approach TEXT NOT NULL CHECK (client_approach IN ('webview', 'native', 'sdui')),
    device_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS point_ledger (
    id BIGSERIAL PRIMARY KEY,
    entry_uuid UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    user_id TEXT NOT NULL REFERENCES users(id),
    entry_type TEXT NOT NULL CHECK (entry_type IN ('SEED', 'DRAW_DEBIT', 'REWARD_CREDIT')),
    delta_points BIGINT NOT NULL CHECK (delta_points <> 0),
    reference_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_point_ledger_user_id ON point_ledger (user_id, id);

-- Memastikan ledger bersifat strictly append-only di level database
CREATE OR REPLACE FUNCTION prevent_ledger_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION '% is strictly append-only; % is not allowed', TG_TABLE_NAME, TG_OP;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_point_ledger_immutable ON point_ledger;
CREATE TRIGGER trg_point_ledger_immutable
BEFORE UPDATE OR DELETE ON point_ledger
FOR EACH ROW EXECUTE FUNCTION prevent_ledger_mutation();

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    action TEXT NOT NULL,
    outcome TEXT NOT NULL,
    table_version TEXT,
    draw_id UUID,
    idempotency_key TEXT,
    client_approach TEXT,
    automation_flag BOOLEAN NOT NULL DEFAULT FALSE,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DROP TRIGGER IF EXISTS trg_audit_logs_immutable ON audit_logs;
CREATE TRIGGER trg_audit_logs_immutable
BEFORE UPDATE OR DELETE ON audit_logs
FOR EACH ROW EXECUTE FUNCTION prevent_ledger_mutation();
