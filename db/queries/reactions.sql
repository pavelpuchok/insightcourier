-- name: CreateReaction :exec
INSERT INTO reactions (source_item_id, type, created_at) VALUES (
    $1, $2, CURRENT_TIMESTAMP
);
