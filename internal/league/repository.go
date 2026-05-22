package league

import (
	"context"

	"github.com/Floha0/football-sim/internal/domain"
)

// TeamRepository persists and retrieves teams.
//
// Methods return league.ErrNotFound when a requested entity does not exist.
type TeamRepository interface {
	GetByID(ctx context.Context, id int) (domain.Team, error)

	GetAll(ctx context.Context) ([]domain.Team, error)
}

// MatchRepository persists and retrieves matches.
//
// Methods return ErrNotFound when an entity is missing, and ErrDuplicateFixture
// or ErrInvalidMatch on constraint violations.
type MatchRepository interface {
	Create(ctx context.Context, m domain.Match) (domain.Match, error)

	// All-or-nothing: if any insert fails, none are committed.
	CreateBatch(ctx context.Context, matches []domain.Match) error

	GetByID(ctx context.Context, id int) (domain.Match, error)

	GetByWeek(ctx context.Context, week int) ([]domain.Match, error)

	GetAll(ctx context.Context) ([]domain.Match, error)

	UpdateResult(ctx context.Context, id, homeGoals, awayGoals int) error

	DeleteAll(ctx context.Context) error
}
