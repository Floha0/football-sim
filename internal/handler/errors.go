package handler

import (
	"errors"
	"net/http"

	"github.com/Floha0/football-sim/internal/league"
)

// errorMapping translates a domain error to an HTTP status code and a
// stable error code string. Unknown errors map to 500.
func errorMapping(err error) (int, string) {
	switch {
	case errors.Is(err, league.ErrNotFound):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, league.ErrFixturesAlreadyExist):
		return http.StatusConflict, "fixtures_exist"
	case errors.Is(err, league.ErrSeasonComplete):
		return http.StatusConflict, "season_complete"
	case errors.Is(err, league.ErrDuplicateFixture):
		return http.StatusConflict, "duplicate_fixture"
	case errors.Is(err, league.ErrTeamNameTaken):
		return http.StatusConflict, "team_name_taken"
	case errors.Is(err, league.ErrInvalidMatch):
		return http.StatusBadRequest, "invalid_match"
	default:
		return http.StatusInternalServerError, "internal_error"
	}
}
