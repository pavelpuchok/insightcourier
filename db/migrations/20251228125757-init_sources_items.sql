
-- +migrate Up
CREATE TABLE sources_items (
  source_item_id SERIAL PRIMARY KEY,
  source_id INT REFERENCES sources(source_id),
  url TEXT,

  title TEXT,
  text_content TEXT,
  excerpt TEXT,
  language TEXT,
  published_at TIMESTAMPTZ,

  created_at TIMESTAMP NOT NULL
);

-- +migrate Down
DROP TABLE sources_items;
