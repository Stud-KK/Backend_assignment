// Package handlers contains all HTTP handler functions.
// Handlers are kept thin — they validate input, call the relevant
// store/service method, and write a JSON response. Business logic
// lives in the packages they call (ratelimiter, products).
package handlers

import (
	"encoding/json"
	"net/http"
)

// WriteJSON serialises v as JSON and writes it with the given HTTP status.
// It always sets Content-Type: application/json.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// WriteError writes a standard {"error": "..."} JSON body with the given status.
// Using a consistent error shape makes it easier for API consumers to handle errors.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// decodeJSON is a small helper that decodes the request body into v.
// Keeping this centralised means we can easily add logging or size
// limits here in the future without touching every handler.
func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
