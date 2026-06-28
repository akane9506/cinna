-- name: ListRecentAgentMemory :many
SELECT *
FROM (
    SELECT *
    FROM agent_memory
    WHERE telegram_user_id = sqlc.arg(telegram_user_id)
    ORDER BY id DESC
    LIMIT sqlc.arg(max_length)
) recent
ORDER BY id ASC;

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

-- name: PruneAgentMemory :exec
DELETE FROM agent_memory AS am
WHERE am.telegram_user_id = sqlc.arg(telegram_user_id)
    AND am.id NOT IN (
        SELECT keep.id
        FROM agent_memory AS keep
        WHERE keep.telegram_user_id = sqlc.arg(telegram_user_id)
        ORDER BY keep.id DESC
        LIMIT sqlc.arg(keep_count)
);
