-- name: CreateSourceItem :one
INSERT INTO sources_items (source_id, url, title, text_content, excerpt, language, published_at, created_at) VALUES($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP) RETURNING source_item_id;
