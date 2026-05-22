package postgres

import (
	"context"
	"fmt"

	"github.com/Floha0/football-sim/internal/domain"
	"github.com/Floha0/football-sim/internal/league"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MatchRepo implements league.MatchRepository
type MatchRepo struct {
	pool *pgxpool.Pool
}

// NewMatchRepo constructs a MatchRepo backed by the given pool.
func NewMatchRepo(pool *pgxpool.Pool) *MatchRepo {
	return &MatchRepo{pool: pool}
}

// Create inserts a single match and returns the persisted entity with its
// database-assigned ID.
func (r *MatchRepo) Create(ctx context.Context, m domain.Match) (domain.Match, error) {
	const query = `
		INSERT INTO matches (week, home_team_id, away_team_id, home_goals, away_goals, played)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		m.Week, m.HomeTeamID, m.AwayTeamID, m.HomeGoals, m.AwayGoals, m.Played,
	).Scan(&m.ID)
	if err != nil {
		return domain.Match{}, fmt.Errorf("create match: %w", translateError(err))
	}
	return m, nil
}

// CreateBatch inserts multiple matches atomically. Either all rows are
// inserted, or none are.
func (r *MatchRepo) CreateBatch(ctx context.Context, matches []domain.Match) error {
	if len(matches) == 0 {
		return nil
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	// Defer rollback to safely roll back.
	defer func() { _ = tx.Rollback(ctx) }()

	const query = `
		INSERT INTO matches (week, home_team_id, away_team_id, home_goals, away_goals, played)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	batch := &pgx.Batch{}
	for _, m := range matches {
		batch.Queue(query, m.Week, m.HomeTeamID, m.AwayTeamID, m.HomeGoals, m.AwayGoals, m.Played)
	}

	br := tx.SendBatch(ctx, batch)
	for i := 0; i < len(matches); i++ {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			return fmt.Errorf("batch insert match %d: %w", i, translateError(err))
		}
	}
	if err := br.Close(); err != nil {
		return fmt.Errorf("close batch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// GetByID retrieves a single match by ID.
func (r *MatchRepo) GetByID(ctx context.Context, id int) (domain.Match, error) {
	const query = `
		SELECT id, week, home_team_id, away_team_id, home_goals, away_goals, played
		FROM matches
		WHERE id = $1
	`
	var m domain.Match
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.Week, &m.HomeTeamID, &m.AwayTeamID,
		&m.HomeGoals, &m.AwayGoals, &m.Played,
	)
	if err != nil {
		return domain.Match{}, fmt.Errorf("get match by id %d: %w", id, translateError(err))
	}
	return m, nil
}

// GetByWeek retrieves all matches scheduled for the given week.
func (r *MatchRepo) GetByWeek(ctx context.Context, week int) ([]domain.Match, error) {
	const query = `
		SELECT id, week, home_team_id, away_team_id, home_goals, away_goals, played
		FROM matches
		WHERE week = $1
		ORDER BY id
	`
	return r.queryMatches(ctx, query, week)
}

// GetAll retrieves every match across all weeks.
func (r *MatchRepo) GetAll(ctx context.Context) ([]domain.Match, error) {
	const query = `
		SELECT id, week, home_team_id, away_team_id, home_goals, away_goals, played
		FROM matches
		ORDER BY week, id
	`
	return r.queryMatches(ctx, query)
}

// UpdateResult records the score of a played match. The played flag is
// always set to true on a successful update.
func (r *MatchRepo) UpdateResult(ctx context.Context, id, homeGoals, awayGoals int) error {
	const query = `
		UPDATE matches
		SET home_goals = $2,
		    away_goals = $3,
		    played     = TRUE,
		    updated_at = NOW()
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, query, id, homeGoals, awayGoals)
	if err != nil {
		return fmt.Errorf("update match %d: %w", id, translateError(err))
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update match %d: %w", id, league.ErrNotFound)
	}
	return nil
}

// DeleteAll removes every match. Used during league reset.
func (r *MatchRepo) DeleteAll(ctx context.Context) error {
	const query = `DELETE FROM matches`
	if _, err := r.pool.Exec(ctx, query); err != nil {
		return fmt.Errorf("delete all matches: %w", err)
	}
	return nil
}

// queryMatches is a private helper that runs a SELECT and scans into a slice.
// Used by GetByWeek and GetAll to avoid duplicating the scan loop.
func (r *MatchRepo) queryMatches(ctx context.Context, query string, args ...any) ([]domain.Match, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query matches: %w", err)
	}
	defer rows.Close()

	matches := make([]domain.Match, 0, 12)
	for rows.Next() {
		var m domain.Match
		if err := rows.Scan(
			&m.ID, &m.Week, &m.HomeTeamID, &m.AwayTeamID,
			&m.HomeGoals, &m.AwayGoals, &m.Played,
		); err != nil {
			return nil, fmt.Errorf("scan match: %w", err)
		}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate matches: %w", err)
	}
	return matches, nil
}
