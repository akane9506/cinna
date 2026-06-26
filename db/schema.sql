-- Table for the allowed users
CREATE TABLE IF NOT EXISTS allowed_users (
    telegram_user_id BIGINT PRIMARY KEY
        CHECK (telegram_user_id > 0),

    role TEXT NOT NULL DEFAULT 'member'
        CHECK (role IN ('admin','member')),

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Table for the shopping list
-- If you modify this part, also change internal/app/agent/prompt/shopping_task_planner.md
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'shopping_category') THEN
        CREATE TYPE shopping_category AS ENUM (
            'grocery',
            'pharmacy',
            'pet',
            'toy',
            'stationery',
            'other'
        );
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS shopping_list (
    id BIGSERIAL PRIMARY KEY,

    telegram_user_id BIGINT NOT NULL
        REFERENCES allowed_users(telegram_user_id)
        ON DELETE CASCADE,

    name TEXT NOT NULL
        CHECK (length(btrim(name)) > 0),

    category shopping_category NOT NULL DEFAULT 'other',

    create_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS shopping_list_user_idx
    ON shopping_list (telegram_user_id);

CREATE INDEX IF NOT EXISTS shopping_list_user_category_idx
    ON shopping_list (telegram_user_id, category);

CREATE UNIQUE INDEX IF NOT EXISTS shopping_list_user_name_unique_idx
    ON shopping_list (
        telegram_user_id,
        lower(btrim(name))
    );
