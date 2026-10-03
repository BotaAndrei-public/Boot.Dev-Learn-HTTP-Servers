-- +goose Up
CREATE TABLE chirps(
    id UUID PRIMARY KEY,
    created_at TIMESTAMP not NULL,
    updated_at TIMESTAMP not NULL,
    body TEXT not NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE chirps;
