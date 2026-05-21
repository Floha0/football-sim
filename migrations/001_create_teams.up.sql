CREATE TABLE teams (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    power       INTEGER NOT NULL CHECK (power BETWEEN 1 AND 100),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);