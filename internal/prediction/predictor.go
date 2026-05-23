// Package prediction estimates championship probabilities through repeated
// simulation of the remainder of the season.
package prediction

import (
	"sort"

	"github.com/Floha0/football-sim/internal/config"
	"github.com/Floha0/football-sim/internal/domain"
	"github.com/Floha0/football-sim/internal/league"
	"github.com/Floha0/football-sim/internal/simulation"
)

// MonteCarloPredictor estimates each team's championship odds by playing
// out the unplayed remainder of the season many times and counting how
// often each team finishes first.
type MonteCarloPredictor struct {
	cfg       config.PredictionConfig
	simulator simulation.MatchSimulator
}

// NewMonteCarloPredictor constructs a predictor backed by the given
// simulator. The simulator should produce non-deterministic results
// across iterations (i.e. seeded with randomness, not a fixed seed).
func NewMonteCarloPredictor(cfg config.PredictionConfig, sim simulation.MatchSimulator) *MonteCarloPredictor {
	return &MonteCarloPredictor{cfg: cfg, simulator: sim}
}

// Predict runs N simulations of the unplayed remainder and returns the
// proportion of runs in which each team finished first. The currentWeek
// argument is included in the result for downstream consumers.
func (p *MonteCarloPredictor) Predict(
	teams []domain.Team,
	matches []domain.Match,
	currentWeek int,
) []domain.Prediction {
	titleWins := make(map[int]int, len(teams))
	for _, t := range teams {
		titleWins[t.ID] = 0
	}

	teamByID := indexTeams(teams)

	for i := 0; i < p.cfg.MonteCarloIterations; i++ {
		// Copy matches so each iteration starts from the real state.
		snapshot := make([]domain.Match, len(matches))
		copy(snapshot, matches)

		// Simulate every unplayed match in this iteration.
		for j := range snapshot {
			if snapshot[j].Played {
				continue
			}
			home := teamByID[snapshot[j].HomeTeamID]
			away := teamByID[snapshot[j].AwayTeamID]
			hg, ag := p.simulator.Simulate(home, away)
			snapshot[j].HomeGoals = hg
			snapshot[j].AwayGoals = ag
			snapshot[j].Played = true
		}

		standings := league.ComputeStandings(teams, snapshot)
		if len(standings) > 0 {
			titleWins[standings[0].TeamID]++
		}
	}

	predictions := make([]domain.Prediction, 0, len(teams))
	for _, t := range teams {
		predictions = append(predictions, domain.Prediction{
			TeamID: t.ID,
			Chance: float64(titleWins[t.ID]) / float64(p.cfg.MonteCarloIterations),
			Week:   currentWeek,
		})
	}

	// Order by descending chance for nicer output.
	sort.SliceStable(predictions, func(i, j int) bool {
		return predictions[i].Chance > predictions[j].Chance
	})

	return predictions
}

func indexTeams(teams []domain.Team) map[int]domain.Team {
	m := make(map[int]domain.Team, len(teams))
	for _, t := range teams {
		m[t.ID] = t
	}
	return m
}
