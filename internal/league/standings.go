package league

import (
	"sort"

	"github.com/Floha0/football-sim/internal/domain"
)

// ComputeStandings aggregates played matches into a sorted league table.
//
// Standings are sorted using Premier League tiebreaker order:
//  1. Points (descending)
//  2. Goal Difference (descending)
//  3. Goals For (descending)
//  4. Head-to-head points among tied teams (descending)
//
// Teams with no played matches still appear in the table with zero stats.
func ComputeStandings(teams []domain.Team, matches []domain.Match) []domain.Standing {
	// Initialize an empty standing per team so the result always includes
	// every team, even those without played matches.
	byID := make(map[int]*domain.Standing, len(teams))
	for _, t := range teams {
		byID[t.ID] = &domain.Standing{TeamID: t.ID}
	}

	// Aggregate played matches into per-team running totals.
	for _, m := range matches {
		if !m.Played {
			continue
		}
		home := byID[m.HomeTeamID]
		away := byID[m.AwayTeamID]
		if home == nil || away == nil {
			continue // orphan match, skip
		}

		home.Played++
		away.Played++
		home.GoalsFor += m.HomeGoals
		home.GoalsAgainst += m.AwayGoals
		away.GoalsFor += m.AwayGoals
		away.GoalsAgainst += m.HomeGoals

		switch {
		case m.HomeGoals > m.AwayGoals:
			home.Wins++
			home.Points += 3
			away.Losses++
		case m.HomeGoals < m.AwayGoals:
			away.Wins++
			away.Points += 3
			home.Losses++
		default:
			home.Draws++
			away.Draws++
			home.Points++
			away.Points++
		}
	}

	out := make([]domain.Standing, 0, len(byID))
	for _, s := range byID {
		out = append(out, *s)
	}

	sortStandings(out, matches)
	return out
}

// sortStandings orders standings by the four-level Premier League tiebreaker.
func sortStandings(standings []domain.Standing, matches []domain.Match) {
	sort.SliceStable(standings, func(i, j int) bool {
		a, b := standings[i], standings[j]

		// 1. Points (descending)
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		// 2. Goal Difference (descending)
		if a.GoalDiff() != b.GoalDiff() {
			return a.GoalDiff() > b.GoalDiff()
		}
		// 3. Goals For (descending)
		if a.GoalsFor != b.GoalsFor {
			return a.GoalsFor > b.GoalsFor
		}
		// 4. Head-to-head points (descending)
		return headToHeadPoints(a.TeamID, b.TeamID, matches) >
			headToHeadPoints(b.TeamID, a.TeamID, matches)
	})
}

// headToHeadPoints returns the total points teamA earned against teamB.
func headToHeadPoints(teamA, teamB int, matches []domain.Match) int {
	points := 0
	for _, m := range matches {
		if !m.Played {
			continue
		}
		if m.HomeTeamID == teamA && m.AwayTeamID == teamB {
			switch {
			case m.HomeGoals > m.AwayGoals:
				points += 3
			case m.HomeGoals == m.AwayGoals:
				points++
			}
		} else if m.HomeTeamID == teamB && m.AwayTeamID == teamA {
			switch {
			case m.AwayGoals > m.HomeGoals:
				points += 3
			case m.HomeGoals == m.AwayGoals:
				points++
			}
		}
	}
	return points
}
