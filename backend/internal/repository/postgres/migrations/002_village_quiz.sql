ALTER TABLE point_ledger DROP CONSTRAINT IF EXISTS point_ledger_entry_type_check;
ALTER TABLE point_ledger ADD CONSTRAINT point_ledger_entry_type_check
    CHECK (entry_type IN ('SEED', 'DRAW_DEBIT', 'REWARD_CREDIT', 'QUIZ_REWARD'));

CREATE TABLE IF NOT EXISTS village_quizzes (
    id TEXT PRIMARY KEY,
    building_name TEXT NOT NULL,
    title TEXT NOT NULL,
    pin_color TEXT NOT NULL CHECK (pin_color IN ('blue', 'orange', 'red')),
    x_ratio DOUBLE PRECISION NOT NULL,
    y_ratio DOUBLE PRECISION NOT NULL,
    reward_points BIGINT NOT NULL CHECK (reward_points > 0),
    question TEXT NOT NULL,
    options JSONB NOT NULL,
    correct_index INT NOT NULL CHECK (correct_index >= 0),
    explanation TEXT NOT NULL,
    display_order INT NOT NULL
);

CREATE TABLE IF NOT EXISTS village_quiz_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id),
    quiz_id TEXT NOT NULL REFERENCES village_quizzes(id),
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    selected_index INT NOT NULL,
    is_correct BOOLEAN NOT NULL,
    reward_points BIGINT NOT NULL CHECK (reward_points >= 0),
    balance_before BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    client_approach TEXT NOT NULL CHECK (client_approach IN ('webview', 'native', 'sdui')),
    device_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, idempotency_key)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_village_quiz_user_completed
    ON village_quiz_submissions (user_id, quiz_id) WHERE is_correct = TRUE;
