-- name: CreateShoppingListItems :many
WITH input_items AS (
    SELECT
        btrim(item_name) AS name,
        ord
    FROM unnest(sqlc.arg(item_names)::text[]) WITH ORDINALITY AS t(item_name, ord)
    WHERE length(btrim(item_name)) > 0
),
deduped_items AS (
    SELECT DISTINCT ON (lower(name))
        name
    FROM input_items
    ORDER BY lower(name), ord
)
INSERT INTO shopping_list (
    telegram_user_id,
    name,
    category
)
SELECT
    sqlc.arg(telegram_user_id),
    name,
    sqlc.arg(category)::shopping_category
FROM deduped_items
ON CONFLICT (
    telegram_user_id,
    lower(btrim(name))
)
DO UPDATE SET
    category = EXCLUDE.category,
    updated_at = NOW()
RETURNING *;

-- name: ListShoppingListItems :many
SELECT *
FROM shopping_list
WHERE telegram_user_id = $1
ORDER BY category, name;
