-- name: ListIncompleteFeedbacks :many
SELECT
    id,
    content,
    status,
    updated_at
FROM feedbacks
WHERE status IN ('pending', 'in_progress')
ORDER BY updated_at DESC;

-- name: CreateFeedbackItems :many
WITH input_items AS (
    SELECT
        btrim(c.content) AS content,
        c.ord
    FROM unnest(sqlc.arg(contents)::text[]) WITH ORDINALITY AS c(content, ord)
    WHERE length(btrim(c.content)) > 0
),
deduped_items AS (
    SELECT DISTINCT ON (lower(content))
        content
    FROM input_items
    ORDER BY lower(content), ord
)
INSERT INTO feedbacks (
    telegram_user_id,
    content
)
SELECT
    sqlc.arg(telegram_user_id),
    content
FROM deduped_items
ON CONFLICT (
    telegram_user_id,
    lower(btrim(content))
)
DO UPDATE SET
    status = CASE
        WHEN feedbacks.status = 'completed' THEN 'pending'
        ELSE feedbacks.status
    END,
    updated_at = NOW()
RETURNING *;
