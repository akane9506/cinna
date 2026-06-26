-- name: CreateShoppingListItems :many
WITH input_items AS (
    SELECT
        btrim(n.item_name) AS name,
        c.category::shopping_category AS category,
        n.ord
    FROM unnest(sqlc.arg(item_names)::text[]) WITH ORDINALITY AS n(item_name, ord)
    JOIN unnest(sqlc.arg(categories)::text[]) WITH ORDINALITY AS c(category, ord)
        USING (ord)
    WHERE length(btrim(n.item_name)) > 0
),
deduped_items AS (
    SELECT DISTINCT ON (lower(name))
        name,
        category
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
    category
FROM deduped_items
ON CONFLICT (
    telegram_user_id,
    lower(btrim(name))
)
DO UPDATE SET
    category = EXCLUDED.category,
    updated_at = NOW()
RETURNING *;

-- name: ListShoppingListItems :many
SELECT *
FROM shopping_list
WHERE telegram_user_id = sqlc.arg(telegram_user_id)
ORDER BY category, name;

-- name: RemoveShoppingListItems :many
WITH input_ids AS (
    SELECT DISTINCT item_id
    FROM unnest(sqlc.arg(item_ids)::bigint[]) AS t(item_id)
)
DELETE FROM shopping_list
USING input_ids
WHERE shopping_list.telegram_user_id = sqlc.arg(telegram_user_id)
    AND shopping_list.id = input_ids.item_id
RETURNING shopping_list.*;

-- name: UpdateShoppingListItems :many
WITH input_items AS (
    SELECT
        i.item_id,
        btrim(n.item_name) AS name,
        c.category::shopping_category AS category,
        i.ord
    FROM unnest(sqlc.arg(item_ids)::bigint[]) WITH ORDINALITY AS i(item_id, ord)
    JOIN unnest(sqlc.arg(item_names)::text[]) WITH ORDINALITY AS n(item_name, ord)
        USING (ord)
    JOIN unnest(sqlc.arg(categories)::text[]) WITH ORDINALITY AS c(category, ord)
        USING (ord)
    WHERE i.item_id > 0
        AND length(btrim(n.item_name)) > 0
),
deduped_items AS(
    SELECT DISTINCT ON (item_id)
        item_id,
        name,
        category
    FROM input_items
    ORDER BY item_id, ord
)
UPDATE shopping_list
SET
    name = deduped_items.name,
    category = deduped_items.category,
    updated_at = NOW()
FROM deduped_items
WHERE shopping_list.telegram_user_id = sqlc.arg(telegram_user_id)
    AND shopping_list.id = deduped_items.item_id
RETURNING shopping_list.*;
