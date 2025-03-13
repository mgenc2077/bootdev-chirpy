-- +goose Up
CREATE TABLE chirps(
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    body TEXT NOT NULL,
    user_id UUID REFERENCES users(id)  ON DELETE CASCADE NOT NULL
);

-- +goose Down
DROP TABLE chirps;
