package domain

// Match represents a single fixture between two teams in a given week.
// HomeGoals and AwayGoals are meaningful only when Played is true.
type Match struct {
	ID         int
	Week       int
	HomeTeamID int
	AwayTeamID int
	HomeGoals  int
	AwayGoals  int
	Played     bool
}

type Result int

const (
	ResultUnknown Result = iota
	ResultHomeWin
	ResultDraw
	ResultAwayWin
)

// Outcome returns the match result from the home team's perspective.
// Returns ResultUnknown if the match hasn't been played.
func (m Match) Outcome() Result {
	if !m.Played {
		return ResultUnknown
	}
	switch {
	case m.HomeGoals > m.AwayGoals:
		return ResultHomeWin
	case m.HomeGoals < m.AwayGoals:
		return ResultAwayWin
	default:
		return ResultDraw
	}
}
