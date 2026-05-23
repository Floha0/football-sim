package handler

import (
	"net/http"
)

// RegisterRoutes wires API endpoints onto the provided mux.
//
// A method/path combination that doesn't appear here results in either a
// 405 Method Not Allowed (path matches but method doesn't) or a 404
// Not Found (path doesn't match anything), routed through notFoundHandler.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("GET /health", h.health)

	// League operations.
	mux.HandleFunc("GET  /api/standings", h.getStandings)
	mux.HandleFunc("GET  /api/matches", h.getMatches)
	mux.HandleFunc("POST /api/play-week", h.playWeek)
	mux.HandleFunc("POST /api/play-all", h.playAll)
	mux.HandleFunc("GET  /api/predictions", h.getPredictions)
	mux.HandleFunc("POST /api/reset", h.resetLeague)

	// Match editing (path parameter for ID).
	mux.HandleFunc("PUT /api/matches/{id}", h.editMatch)

	// Catch-all for anything not matched. Returns JSON 404 to keep the
	// API consistent. Without this, unknown routes return text/plain.
	mux.HandleFunc("/", h.notFound)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// notFound returns a consistent JSON 404 for any unmatched route.
func (h *Handler) notFound(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusNotFound, ErrorResponse{
		Error: "not found",
		Code:  "not_found",
	})
}
