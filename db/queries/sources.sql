-- name: GetSourceLastFetchedAtByName :one
SELECT last_fetched_at FROM sources WHERE name = $1;

-- name: CreateSource :one
INSERT INTO sources (name, created_at, updated_at)
VALUES ( $1, $2, CURRENT_TIMESTAMP )
RETURNING source_id;

-- name: SetSourceLastFetchedAtByName :exec
UPDATE SOURCES
  SET last_fetched_at = $2, updated_at = CURRENT_TIMESTAMP
WHERE name = $1;
