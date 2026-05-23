package league

import (
	"fmt"

	"github.com/Floha0/football-sim/internal/domain"
)

// GenerateRoundRobin produces a double round-robin schedule for the given
// teams. Each pair of teams plays twice — once home, once away.
//
// For N teams (N even), the schedule has 2*(N-1) weeks with N/2 matches per
// week. The algorithm uses the "circle method", ensuring every team plays
// every other exactly once per half.
//
// Returns ErrInvalidMatch if the team count is odd or less than two.
func GenerateRoundRobin(teams []domain.Team) ([]domain.Match, error) {
	n := len(teams)
	if n < 2 || n%2 != 0 {
		return nil, fmt.Errorf("%w: need an even number of teams, got %d", ErrInvalidMatch, n)
	}

	// First half of the season: each pair plays once.
	firstHalf := generateFirstHalf(teams)

	// Second half: same fixtures with home/away swapped, offset by (n-1) weeks.
	weeksPerHalf := n - 1
	matches := make([]domain.Match, 0, len(firstHalf)*2)
	matches = append(matches, firstHalf...)

	for _, m := range firstHalf {
		matches = append(matches, domain.Match{
			Week:       m.Week + weeksPerHalf,
			HomeTeamID: m.AwayTeamID, // swap home/away
			AwayTeamID: m.HomeTeamID,
		})
	}

	return matches, nil
}

// generateFirstHalf implements the circle method for round-robin scheduling.
func generateFirstHalf(teams []domain.Team) []domain.Match {
	n := len(teams)
	weeksPerHalf := n - 1
	matchesPerWeek := n / 2

	rotation := make([]domain.Team, n)
	copy(rotation, teams)

	matches := make([]domain.Match, 0, weeksPerHalf*matchesPerWeek)

	for week := 1; week <= weeksPerHalf; week++ {
		for i := 0; i < matchesPerWeek; i++ {
			home := rotation[i]
			away := rotation[n-1-i]

			// Alternate home/away each week for the fixed team to balance fairness.
			if week%2 == 0 && i == 0 {
				home, away = away, home
			}

			matches = append(matches, domain.Match{
				Week:       week,
				HomeTeamID: home.ID,
				AwayTeamID: away.ID,
			})
		}

		// Rotate everyone except the fixed team (index 0).
		// Element at index 1 moves to the end; the rest shift left by one.
		last := rotation[1]
		copy(rotation[1:], rotation[2:])
		rotation[n-1] = last
	}

	return matches
}
