// Package handler exposes the league service over HTTP.
//
// The DTOs in this file are deliberately separate from domain types. They
// represent the API's wire contract: their field names, ordering, and
// optional fields can change independently of internal domain shapes.
package handler

import "github.com/Floha0/football-sim/internal/domain"

// TeamDTO is the API representation of a team.
type TeamDTO struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Power int    `json:"power"`
}

// MatchDTO is the API representation of a match. HomeTeam and AwayTeam
// names are denormalized for client convenience.
type MatchDTO struct {
	ID         int    `json:"id"`
	Week       int    `json:"week"`
	HomeTeamID int    `json:"homeTeamId"`
	AwayTeamID int    `json:"awayTeamId"`
	HomeTeam   string `json:"homeTeam,omitempty"`
	AwayTeam   string `json:"awayTeam,omitempty"`
	HomeGoals  int    `json:"homeGoals"`
	AwayGoals  int    `json:"awayGoals"`
	Played     bool   `json:"played"`
}

// StandingDTO is one row of the league table. GoalDiff is included
// explicitly (it's computed on the domain side) so clients don't
// have to recalculate.
type StandingDTO struct {
	Position     int    `json:"position"`
	TeamID       int    `json:"teamId"`
	TeamName     string `json:"teamName,omitempty"`
	Played       int    `json:"played"`
	Wins         int    `json:"wins"`
	Draws        int    `json:"draws"`
	Losses       int    `json:"losses"`
	GoalsFor     int    `json:"goalsFor"`
	GoalsAgainst int    `json:"goalsAgainst"`
	GoalDiff     int    `json:"goalDiff"`
	Points       int    `json:"points"`
}

// PredictionDTO is the championship probability for one team at a given week.
type PredictionDTO struct {
	TeamID   int     `json:"teamId"`
	TeamName string  `json:"teamName,omitempty"`
	Chance   float64 `json:"chance"`
	Week     int     `json:"week"`
}

// EditMatchRequest is the body for PUT /api/matches/{id}.
type EditMatchRequest struct {
	HomeGoals int `json:"homeGoals"`
	AwayGoals int `json:"awayGoals"`
}

// PlayWeekResponse describes a single completed week.
type PlayWeekResponse struct {
	Week    int        `json:"week"`
	Matches []MatchDTO `json:"matches"`
}

// PlayAllResponse describes all weeks played in one call.
type PlayAllResponse struct {
	Weeks []PlayWeekResponse `json:"weeks"`
}

// ErrorResponse is the consistent envelope for all error responses.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// --- Domain → DTO mapping ---

// teamToDTO converts a domain Team to its API representation.
func teamToDTO(t domain.Team) TeamDTO {
	return TeamDTO{ID: t.ID, Name: t.Name, Power: t.Power}
}

// matchToDTO converts a domain Match, optionally enriching with team names
// from the given lookup map. Pass nil to skip name lookup.
func matchToDTO(m domain.Match, names map[int]string) MatchDTO {
	dto := MatchDTO{
		ID:         m.ID,
		Week:       m.Week,
		HomeTeamID: m.HomeTeamID,
		AwayTeamID: m.AwayTeamID,
		HomeGoals:  m.HomeGoals,
		AwayGoals:  m.AwayGoals,
		Played:     m.Played,
	}
	if names != nil {
		dto.HomeTeam = names[m.HomeTeamID]
		dto.AwayTeam = names[m.AwayTeamID]
	}
	return dto
}

// matchesToDTOs converts a slice of matches with name enrichment.
func matchesToDTOs(matches []domain.Match, names map[int]string) []MatchDTO {
	out := make([]MatchDTO, len(matches))
	for i, m := range matches {
		out[i] = matchToDTO(m, names)
	}
	return out
}

// standingsToDTOs converts standings with position numbers and team names.
func standingsToDTOs(standings []domain.Standing, names map[int]string) []StandingDTO {
	out := make([]StandingDTO, len(standings))
	for i, s := range standings {
		out[i] = StandingDTO{
			Position:     i + 1,
			TeamID:       s.TeamID,
			TeamName:     names[s.TeamID],
			Played:       s.Played,
			Wins:         s.Wins,
			Draws:        s.Draws,
			Losses:       s.Losses,
			GoalsFor:     s.GoalsFor,
			GoalsAgainst: s.GoalsAgainst,
			GoalDiff:     s.GoalDiff(),
			Points:       s.Points,
		}
	}
	return out
}

// predictionsToDTOs converts predictions with team names.
func predictionsToDTOs(preds []domain.Prediction, names map[int]string) []PredictionDTO {
	out := make([]PredictionDTO, len(preds))
	for i, p := range preds {
		out[i] = PredictionDTO{
			TeamID:   p.TeamID,
			TeamName: names[p.TeamID],
			Chance:   p.Chance,
			Week:     p.Week,
		}
	}
	return out
}
