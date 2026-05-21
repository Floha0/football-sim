CREATE TABLE matches (
    id            SERIAL PRIMARY KEY,
    week          INTEGER NOT NULL CHECK (week >= 1),
    home_team_id  INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    away_team_id  INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    home_goals    INTEGER NOT NULL DEFAULT 0 CHECK (home_goals >= 0),
    away_goals    INTEGER NOT NULL DEFAULT 0 CHECK (away_goals >= 0),
    played        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- A team can't play itself
    CONSTRAINT no_self_match CHECK (home_team_id <> away_team_id),

    -- Prevent same fixtures
    CONSTRAINT unique_fixture_per_week UNIQUE (week, home_team_id, away_team_id)
);

-- Indexes for common queries
CREATE INDEX idx_matches_week ON matches(week);
CREATE INDEX idx_matches_played ON matches(played) WHERE played = FALSE;