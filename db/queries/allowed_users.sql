-- name: IsAllowedUser :one
SELECT EXISTS (
    SELECT 1
    FROM allowed_users
    WHERE telegram_user_id = $1
        AND is_active = TRUE
);

-- name: UpsertAllowedUser :one
INSERT INTO allowed_users (
    telegram_user_id,
    role
) VALUES (
    sqlc.arg(telegram_user_id),
    sqlc.arg(role)
)
ON CONFLICT (telegram_user_id) DO UPDATE SET
    role = EXCLUDED.role,
    is_active = TRUE,
    updated_at = NOW()
RETURNING *;

-- name: ListAllowedUsers :many
SELECT * FROM allowed_users
    ORDER BY updated_at;
