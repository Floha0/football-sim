package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
)

// respondJSON writes a JSON response with the given status code.
// It logs (but does not return) encoding errors — the response is already
// committed by the time encoding fails.
func respondJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode response", "err", err)
	}
}

// respondError writes a JSON error response. It always uses the domain
// error mapping to determine status code, ensuring consistent behavior.
func respondError(w http.ResponseWriter, err error) {
	status, code := errorMapping(err)
	if status >= http.StatusInternalServerError {
		// Don't leak internal error details to clients; log instead.
		slog.Error("handler internal error", "err", err)
		respondJSON(w, status, ErrorResponse{Error: "internal server error", Code: code})
		return
	}
	respondJSON(w, status, ErrorResponse{Error: err.Error(), Code: code})
}

// decodeBody parses the request body into v. Returns a 400-mapping error
// on malformed JSON. Limits body size to prevent abuse.
func decodeBody(r *http.Request, v any) error {
	const maxBodyBytes = 1 << 20 // 1 MiB — plenty for our DTOs
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // reject unexpected fields

	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: %s", errBadRequest, err.Error())
	}
	return nil
}

var errBadRequest = errors.New("bad request")

func sortWeeks(weeks []PlayWeekResponse) {
	sort.SliceStable(weeks, func(i, j int) bool {
		return weeks[i].Week < weeks[j].Week
	})
}
