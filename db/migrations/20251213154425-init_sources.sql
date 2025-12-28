
-- +migrate Up
CREATE TABLE sources (
  source_id SERIAL PRIMARY KEY,
  name VARCHAR(126) NOT NULL UNIQUE,
  last_fetched_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);

-- +migrate Down
DROP TABLE sources;
