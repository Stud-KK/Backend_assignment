package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"source-asia-backend/ratelimiter"
)

func HandleRequest(rl *ratelimiter.RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var body struct {
			UserID  string          `json:"user_id"`
			Payload json.RawMessage `json:"payload"` // accepted as any valid JSON value
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		// user_id must be present and non-empty (whitespace-only is also rejected).
		if strings.TrimSpace(body.UserID) == "" {
			WriteError(w, http.StatusBadRequest, "user_id is required and must be non-empty")
			return
		}

		if len(body.Payload) == 0 {
			WriteError(w, http.StatusBadRequest, "payload is required")
			return
		}

		if !rl.Allow(body.UserID) {
			WriteJSON(w, http.StatusTooManyRequests, map[string]string{
				"error":   "rate limit exceeded",
				"message": fmt.Sprintf("max %d requests per minute allowed for user '%s'", ratelimiter.MaxRequestsPerWindow, body.UserID),
			})
			return
		}

		WriteJSON(w, http.StatusCreated, map[string]any{
			"status":    "accepted",
			"user_id":   body.UserID,
			"timestamp": time.Now().UTC(),
		})
	}
}

func HandleStats(rl *ratelimiter.RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		WriteJSON(w, http.StatusOK, map[string]any{
			"users": rl.Stats(),
		})
	}
}
