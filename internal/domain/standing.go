package domain

// Standing is a single row of the league table for a team after all
// played matches have been accounted for.
type Standing struct {
	TeamID       int
	Played       int
	Wins         int
	Draws        int
	Losses       int
	GoalsFor     int
	GoalsAgainst int
	Points       int
}

// GoalDiff returns the goal difference (GoalsFor - GoalsAgainst).
func (s Standing) GoalDiff() int {
	return s.GoalsFor - s.GoalsAgainst
}
