package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/Floha0/football-sim/internal/domain"
)

// Service is the surface of the league service that handlers consume.
// Defining it here (consumer side) keeps handler code testable with a
// mock and decoupled from league.Service's concrete shape.
type Service interface {
	GenerateFixtures(ctx context.Context) error
	PlayWeek(ctx context.Context) ([]domain.Match, int, error)
	PlayAll(ctx context.Context) (map[int][]domain.Match, error)
	GetStandings(ctx context.Context) ([]domain.Standing, error)
	GetMatchesByWeek(ctx context.Context, week int) ([]domain.Match, error)
	EditMatch(ctx context.Context, id, homeGoals, awayGoals int) error
	Predict(ctx context.Context) ([]domain.Prediction, error)
	Reset(ctx context.Context) error
	GetTeams(ctx context.Context) ([]domain.Team, error)
}

// Handler holds the dependencies needed to serve API requests.
type Handler struct {
	svc Service
}

// New constructs a Handler bound to the given service.
func New(svc Service) *Handler {
	if svc == nil {
		panic("handler.New: svc must be non-nil")
	}
	return &Handler{svc: svc}
}

// teamNames fetches teams and returns a map of ID to name. Used to enrich
// response DTOs with human-readable team names.
func (h *Handler) teamNames(ctx context.Context) (map[int]string, error) {
	teams, err := h.svc.GetTeams(ctx)
	if err != nil {
		return nil, err
	}
	names := make(map[int]string, len(teams))
	for _, t := range teams {
		names[t.ID] = t.Name
	}
	return names, nil
}

// --- Handlers ---

// getStandings returns the current league table.
func (h *Handler) getStandings(w http.ResponseWriter, r *http.Request) {
	standings, err := h.svc.GetStandings(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	names, err := h.teamNames(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, standingsToDTOs(standings, names))
}

// getMatches returns matches for a given week.
func (h *Handler) getMatches(w http.ResponseWriter, r *http.Request) {
	weekStr := r.URL.Query().Get("week")
	if weekStr == "" {
		respondError(w, errBadRequest)
		return
	}
	week, err := strconv.Atoi(weekStr)
	if err != nil || week < 1 {
		respondError(w, errBadRequest)
		return
	}
	matches, err := h.svc.GetMatchesByWeek(r.Context(), week)
	if err != nil {
		respondError(w, err)
		return
	}
	names, err := h.teamNames(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, matchesToDTOs(matches, names))
}

// playWeek advances the league by one week.
func (h *Handler) playWeek(w http.ResponseWriter, r *http.Request) {
	matches, week, err := h.svc.PlayWeek(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	names, err := h.teamNames(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, PlayWeekResponse{
		Week:    week,
		Matches: matchesToDTOs(matches, names),
	})
}

// playAll plays every remaining week.
func (h *Handler) playAll(w http.ResponseWriter, r *http.Request) {
	weeksMap, err := h.svc.PlayAll(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	names, err := h.teamNames(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}

	// Sort weeks numerically for stable output.
	weeks := make([]PlayWeekResponse, 0, len(weeksMap))
	for week, matches := range weeksMap {
		weeks = append(weeks, PlayWeekResponse{
			Week:    week,
			Matches: matchesToDTOs(matches, names),
		})
	}
	sortWeeks(weeks)
	respondJSON(w, http.StatusOK, PlayAllResponse{Weeks: weeks})
}

// getPredictions returns Monte Carlo championship probabilities.
func (h *Handler) getPredictions(w http.ResponseWriter, r *http.Request) {
	preds, err := h.svc.Predict(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	names, err := h.teamNames(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, predictionsToDTOs(preds, names))
}

// resetLeague wipes matches and regenerates fixtures.
func (h *Handler) resetLeague(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Reset(r.Context()); err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// editMatch updates a match's score by ID.
func (h *Handler) editMatch(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		respondError(w, errBadRequest)
		return
	}

	var req EditMatchRequest
	if err := decodeBody(r, &req); err != nil {
		respondError(w, err)
		return
	}

	if err := h.svc.EditMatch(r.Context(), id, req.HomeGoals, req.AwayGoals); err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
