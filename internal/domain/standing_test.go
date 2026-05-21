package domain_test

import (
	"testing"

	"github.com/Floha0/football-sim/internal/domain"
)

func TestMatchOutcome(t *testing.T) {
	tests := []struct {
		name string
		m    domain.Match
		want domain.Result
	}{
		{"not played", domain.Match{Played: false}, domain.ResultUnknown},
		{"home wins", domain.Match{Played: true, HomeGoals: 2, AwayGoals: 1}, domain.ResultHomeWin},
		{"away wins", domain.Match{Played: true, HomeGoals: 0, AwayGoals: 3}, domain.ResultAwayWin},
		{"draw", domain.Match{Played: true, HomeGoals: 1, AwayGoals: 1}, domain.ResultDraw},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.m.Outcome(); got != tc.want {
				t.Errorf("Outcome() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStandingGoalDiff(t *testing.T) {
	tests := []struct {
		name string
		s    domain.Standing
		want int
	}{
		{"positive diff", domain.Standing{GoalsFor: 5, GoalsAgainst: 2}, 3},
		{"negative diff", domain.Standing{GoalsFor: 1, GoalsAgainst: 4}, -3},
		{"zero diff", domain.Standing{GoalsFor: 2, GoalsAgainst: 2}, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.GoalDiff(); got != tc.want {
				t.Errorf("GoalDiff() = %v, want %v", got, tc.want)
			}
		})
	}
}
