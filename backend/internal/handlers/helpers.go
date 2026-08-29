// Package handlers provides HTTP route handlers and request/response encoding helpers.
package handlers

import (
	"encoding/json"
	"net/http"
)

// writeJSON serializes the given data to JSON, writes the appropriate Content-Type header,
// and sends the specified HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// writeError writes a standardized JSON error response: {"error": message}.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// decodeJSON decodes the JSON request body into destination v.
// If decoding fails, it writes a 400 Bad Request error response to w and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "request body cannot be empty")
		return false
	}
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return false
	}

	return true
}
