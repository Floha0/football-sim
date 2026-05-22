package postgres

import (
	"errors"

	"github.com/Floha0/football-sim/internal/league"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// translateError maps low-level pg errors to domain-level errors so that
// callers never depend on database internals. Unknown errors pass through
// unchanged.
func translateError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return league.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation: // 23505
			switch pgErr.ConstraintName {
			case "teams_name_key":
				return league.ErrTeamNameTaken
			case "unique_fixture_per_week":
				return league.ErrDuplicateFixture
			}
		case pgerrcode.CheckViolation: // 23514
			return league.ErrInvalidMatch
		case pgerrcode.ForeignKeyViolation: // 23503
			return league.ErrNotFound
		}
	}

	return err // unknown error, pass through
}
