-- name: IsAdminUser :one
SELECT EXISTS (
    SELECT 1
    FROM allowed_users
    WHERE telegram_user_id = sqlc.arg(telegram_user_id)
        AND role = 'admin'
        AND is_active = TRUE
);

-- name: IsAllowedUser :one
SELECT EXISTS (
    SELECT 1
    FROM allowed_users
    WHERE telegram_user_id = sqlc.arg(telegram_user_id)
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
ON CONFLICT (telegram_user_id) DO NOTHING
RETURNING *;

-- name: ListAllowedUsers :many
SELECT * FROM allowed_users
    ORDER BY updated_at;
