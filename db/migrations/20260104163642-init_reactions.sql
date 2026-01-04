
-- +migrate Up
CREATE TYPE reactions_type AS ENUM ('like', 'dislike');

CREATE TABLE reactions (
    source_item_id INT REFERENCES sources_items (source_item_id) NOT NULL,
    type REACTIONS_TYPE NOT NULL,
    created_at TIMESTAMP NOT NULL,

    UNIQUE (source_item_id)
);

-- +migrate Down
DROP TABLE reactions;

DROP TYPE REACTIONS_TYPE;
