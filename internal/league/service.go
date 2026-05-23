package league

import (
	"context"
	"errors"
	"fmt"

	"github.com/Floha0/football-sim/internal/domain"
	"github.com/Floha0/football-sim/internal/simulation"
)

// Predictor estimates championship probabilities for each team based on
// current standings and remaining unplayed matches.
type Predictor interface {
	Predict(teams []domain.Team, matches []domain.Match, currentWeek int) []domain.Prediction
}

// Service orchestrates the football league lifecycle: fixture generation,
// match simulation, standings computation, and championship prediction.
type Service struct {
	teams     TeamRepository
	matches   MatchRepository
	simulator simulation.MatchSimulator
	predictor Predictor
}

// NewService constructs a Service with the given dependencies.
// All arguments must be non-nil; passing nil panics, since a Service
// without its collaborators cannot function.
func NewService(
	teams TeamRepository,
	matches MatchRepository,
	simulator simulation.MatchSimulator,
	predictor Predictor,
) *Service {
	if teams == nil || matches == nil || simulator == nil || predictor == nil {
		panic("league.NewService: all dependencies must be non-nil")
	}
	return &Service{
		teams:     teams,
		matches:   matches,
		simulator: simulator,
		predictor: predictor,
	}
}

func (s *Service) GetTeams(ctx context.Context) ([]domain.Team, error) {
	return s.teams.GetAll(ctx)
}

// GenerateFixtures creates the full round-robin schedule for all registered
// teams. Returns an error if fixtures already exist, call Reset first to
// regenerate.
func (s *Service) GenerateFixtures(ctx context.Context) error {
	existing, err := s.matches.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("check existing fixtures: %w", err)
	}
	if len(existing) > 0 {
		return ErrFixturesAlreadyExist
	}

	teams, err := s.teams.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("load teams: %w", err)
	}

	fixtures, err := GenerateRoundRobin(teams)
	if err != nil {
		return fmt.Errorf("generate schedule: %w", err)
	}

	if err := s.matches.CreateBatch(ctx, fixtures); err != nil {
		return fmt.Errorf("persist fixtures: %w", err)
	}
	return nil
}

// PlayWeek simulates every unplayed match in the next pending week.
// Returns the played matches and the week number that was advanced.
// Returns ErrSeasonComplete if no unplayed matches remain.
func (s *Service) PlayWeek(ctx context.Context) ([]domain.Match, int, error) {
	all, err := s.matches.GetAll(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("load matches: %w", err)
	}

	nextWeek, found := findNextUnplayedWeek(all)
	if !found {
		return nil, 0, ErrSeasonComplete
	}

	teams, err := s.teams.GetAll(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("load teams: %w", err)
	}
	teamByID := indexTeams(teams)

	played := make([]domain.Match, 0)
	for _, m := range all {
		if m.Week != nextWeek || m.Played {
			continue
		}
		home, okH := teamByID[m.HomeTeamID]
		away, okA := teamByID[m.AwayTeamID]
		if !okH || !okA {
			return nil, 0, fmt.Errorf("orphan match %d: missing team", m.ID)
		}

		homeGoals, awayGoals := s.simulator.Simulate(home, away)
		if err := s.matches.UpdateResult(ctx, m.ID, homeGoals, awayGoals); err != nil {
			return nil, 0, fmt.Errorf("update match %d: %w", m.ID, err)
		}

		m.HomeGoals = homeGoals
		m.AwayGoals = awayGoals
		m.Played = true
		played = append(played, m)
	}

	return played, nextWeek, nil
}

// PlayAll returns the matches played, grouped by week in order.
func (s *Service) PlayAll(ctx context.Context) (map[int][]domain.Match, error) {
	result := make(map[int][]domain.Match)
	for {
		matches, week, err := s.PlayWeek(ctx)
		if errors.Is(err, ErrSeasonComplete) {
			break
		}
		if err != nil {
			return result, err
		}
		result[week] = matches
	}
	return result, nil
}

// GetStandings returns the current league table, sorted by Premier League
// tiebreaker rules.
func (s *Service) GetStandings(ctx context.Context) ([]domain.Standing, error) {
	teams, err := s.teams.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("load teams: %w", err)
	}
	matches, err := s.matches.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("load matches: %w", err)
	}
	return ComputeStandings(teams, matches), nil
}

// GetMatchesByWeek returns all matches in the requested week.
func (s *Service) GetMatchesByWeek(ctx context.Context, week int) ([]domain.Match, error) {
	return s.matches.GetByWeek(ctx, week)
}

// EditMatch overrides the result of an existing match. The match is marked
// played; standings recompute automatically on the next call.
func (s *Service) EditMatch(ctx context.Context, id, homeGoals, awayGoals int) error {
	if homeGoals < 0 || awayGoals < 0 {
		return fmt.Errorf("%w: goals must be non-negative", ErrInvalidMatch)
	}
	return s.matches.UpdateResult(ctx, id, homeGoals, awayGoals)
}

// Predict returns Monte Carlo championship probabilities for each team.
func (s *Service) Predict(ctx context.Context) ([]domain.Prediction, error) {
	teams, err := s.teams.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("load teams: %w", err)
	}
	matches, err := s.matches.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("load matches: %w", err)
	}
	currentWeek := highestPlayedWeek(matches)
	return s.predictor.Predict(teams, matches, currentWeek), nil
}

// Reset removes all matches and regenerates fixtures from scratch.
// Teams are not affected.
func (s *Service) Reset(ctx context.Context) error {
	if err := s.matches.DeleteAll(ctx); err != nil {
		return fmt.Errorf("delete matches: %w", err)
	}
	return s.GenerateFixtures(ctx)
}

// findNextUnplayedWeek returns the lowest week number with at least one
// unplayed match, or false if every match is played.
func findNextUnplayedWeek(matches []domain.Match) (int, bool) {
	week := 0
	found := false
	for _, m := range matches {
		if m.Played {
			continue
		}
		if !found || m.Week < week {
			week = m.Week
			found = true
		}
	}
	return week, found
}

// indexTeams builds a map from team ID to team for O(1) lookup.
func indexTeams(teams []domain.Team) map[int]domain.Team {
	m := make(map[int]domain.Team, len(teams))
	for _, t := range teams {
		m[t.ID] = t
	}
	return m
}

// highestPlayedWeek returns the highest week number with at least one played
// match. Returns 0 if no matches are played.
func highestPlayedWeek(matches []domain.Match) int {
	week := 0
	for _, m := range matches {
		if m.Played && m.Week > week {
			week = m.Week
		}
	}
	return week
}
