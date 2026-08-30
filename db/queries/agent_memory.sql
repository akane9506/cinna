-- name: ListRecentAgentMemory :many
SELECT *
FROM agent_memory
WHERE telegram_user_id = sqlc.arg(telegram_user_id)
ORDER BY created_at ASC, id ASC;

-- name: AppendAgentMemoryBatch :many
WITH input_messages AS (
    SELECT
        r.role,
        c.content,
        r.ord
    FROM unnest(sqlc.arg(roles)::text[]) WITH ORDINALITY AS r(role, ord)
    JOIN unnest(sqlc.arg(contents)::text[]) WITH ORDINALITY AS c(content, ord)
        USING (ord)
    WHERE r.role IN('user', 'assistant')
    AND length(btrim(c.content)) > 0
) INSERT INTO agent_memory (
    telegram_user_id,
    role,
    content
)
SELECT
    sqlc.arg(telegram_user_id),
    role,
    content
FROM input_messages
ORDER BY ord
RETURNING *;

-- name: ReplaceAgentMemory :many
WITH deleted AS (
    DELETE FROM agent_memory
    WHERE telegram_user_id = sqlc.arg(telegram_user_id)
),
input_messages AS (
    SELECT
        r.role,
        c.content,
        r.ord
    FROM unnest(sqlc.arg(roles)::text[]) WITH ORDINALITY AS r(role, ord)
    JOIN unnest(sqlc.arg(contents)::text[]) WITH ORDINALITY AS c(content, ord)
        USING (ord)
    WHERE r.role IN ('user', 'assistant')
        AND length(btrim(c.content)) > 0
)
INSERT INTO agent_memory(
    telegram_user_id,
    role,
    content
)
SELECT
    sqlc.arg(telegram_user_id),
    role,
    content
FROM input_messages
ORDER BY ord
RETURNING *;
