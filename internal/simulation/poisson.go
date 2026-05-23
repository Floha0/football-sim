package simulation

import (
	"math"
	"math/rand/v2"

	"github.com/Floha0/football-sim/internal/config"
	"github.com/Floha0/football-sim/internal/domain"
)

// PoissonSimulator models match outcomes by computing each team's expected
// goals (xG) from team strengths and home advantage, then sampling actual
// goals from a Poisson distribution. This produces a realistic spread of
// scorelines: many 1-0, 2-1, 0-0 results and occasional 4-0 blowouts.
type PoissonSimulator struct {
	cfg config.SimulationConfig
	rng *rand.Rand
}

// NewPoissonSimulator constructs a simulator with its own random source.
// Pass nil for the rng to use a freshly-seeded source.
func NewPoissonSimulator(cfg config.SimulationConfig, rng *rand.Rand) *PoissonSimulator {
	if rng == nil {
		rng = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	}
	return &PoissonSimulator{cfg: cfg, rng: rng}
}

// Simulate produces match goals based on team power and home advantage.
func (s *PoissonSimulator) Simulate(home, away domain.Team) (int, int) {
	homeStrength := float64(home.Power) * s.cfg.HomeAdvantage
	awayStrength := float64(away.Power)
	total := homeStrength + awayStrength

	homeExpected := (homeStrength / total) * s.cfg.AverageTotalGoals
	awayExpected := (awayStrength / total) * s.cfg.AverageTotalGoals

	return s.sample(homeExpected), s.sample(awayExpected)
}

// sample draws a single value from Poisson(lambda) using Knuth's algorithm.
func (s *PoissonSimulator) sample(lambda float64) int {
	if lambda <= 0 {
		return 0
	}
	l := math.Exp(-lambda)
	k := 0
	p := 1.0
	for {
		k++
		p *= s.rng.Float64()
		if p <= l {
			break
		}
		if k > s.cfg.MaxGoalsPerTeam {
			return s.cfg.MaxGoalsPerTeam
		}
	}
	return k - 1
}
