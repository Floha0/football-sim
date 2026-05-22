// Package postgres provides pgx-backed implementations of the league
// repository interfaces.
package postgres

import (
	"context"
	"fmt"

	"github.com/Floha0/football-sim/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TeamRepo implements league.TeamRepository
type TeamRepo struct {
	pool *pgxpool.Pool
}

// NewTeamRepo constructs a TeamRepo backed by the given pool.
func NewTeamRepo(pool *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{pool: pool}
}

// GetByID retrieves a single team by its primary key.
func (r *TeamRepo) GetByID(ctx context.Context, id int) (domain.Team, error) {
	const query = `
		SELECT id, name, power
		FROM teams
		WHERE id = $1
	`
	var t domain.Team
	err := r.pool.QueryRow(ctx, query, id).Scan(&t.ID, &t.Name, &t.Power)
	if err != nil {
		return domain.Team{}, fmt.Errorf("get team by id %d: %w", id, translateError(err))
	}
	return t, nil
}

// GetAll retrieves every team in the league.
func (r *TeamRepo) GetAll(ctx context.Context) ([]domain.Team, error) {
	const query = `
		SELECT id, name, power
		FROM teams
		ORDER BY id
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query teams: %w", err)
	}
	defer rows.Close()

	teams := make([]domain.Team, 0, 4) // 4 teams in this league, sensible preallocation
	for rows.Next() {
		var t domain.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Power); err != nil {
			return nil, fmt.Errorf("scan team: %w", err)
		}
		teams = append(teams, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teams: %w", err)
	}
	return teams, nil
}
