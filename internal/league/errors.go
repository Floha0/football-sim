// Package league contains the league service and the repository interfaces
// it depends on. Concrete implementations live elsewhere (e.g. internal/postgres).
package league

import "errors"

var (
	ErrNotFound = errors.New("resource not found")

	ErrDuplicateFixture = errors.New("duplicate fixture")

	ErrInvalidMatch = errors.New("invalid match")

	ErrTeamNameTaken = errors.New("team name already taken")

	ErrSeasonComplete = errors.New("season complete")

	ErrFixturesAlreadyExist = errors.New("fixtures already exist")
)
