// Package simulation produces match outcomes from team strengths.
//
// The MatchSimulator interface lets the league service stay decoupled from
// any particular simulation strategy. Swapping implementations (e.g. for
// testing) requires no changes to consumer code.
package simulation

import "github.com/Floha0/football-sim/internal/domain"

// MatchSimulator produces a goal count for the home and away teams in a
// single match.
type MatchSimulator interface {
	Simulate(home, away domain.Team) (homeGoals, awayGoals int)
}
