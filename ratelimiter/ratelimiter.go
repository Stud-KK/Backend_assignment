package ratelimiter

import (
	"sync"
	"time"
)

const (
	MaxRequestsPerWindow = 5

	WindowDuration = time.Minute
)

type windowEntry struct {
	count       int       // how many requests accepted so far in this window
	windowStart time.Time // when this window started
}

type RateLimiter struct {
	mu       sync.Mutex
	windows  map[string]*windowEntry // active window per user
	rejected map[string]int          // cumulative rejected requests per user (never resets)
}

// New creates and returns a ready-to-use RateLimiter.
func New() *RateLimiter {
	return &RateLimiter{
		windows:  make(map[string]*windowEntry),
		rejected: make(map[string]int),
	}
}

func (rl *RateLimiter) Allow(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.windows[userID]

	if !exists || now.Sub(entry.windowStart) >= WindowDuration {
		rl.windows[userID] = &windowEntry{count: 1, windowStart: now}
		return true
	}

	// Window is still active — allow if under the limit.
	if entry.count < MaxRequestsPerWindow {
		entry.count++
		return true
	}

	// User has hit the limit; record the rejection and refuse.
	rl.rejected[userID]++
	return false
}

type UserStats struct {
	UserID string `json:"user_id"`

	AcceptedInWindow int `json:"accepted_in_window"`

	RejectedTotal int `json:"rejected_total"`
}

func (rl *RateLimiter) Stats() []UserStats {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	known := make(map[string]bool)
	for id := range rl.windows {
		known[id] = true
	}
	for id := range rl.rejected {
		known[id] = true
	}

	result := []UserStats{}
	for id := range known {
		accepted := 0
		if entry, ok := rl.windows[id]; ok {

			if now.Sub(entry.windowStart) < WindowDuration {
				accepted = entry.count
			}
		}
		result = append(result, UserStats{
			UserID:           id,
			AcceptedInWindow: accepted,
			RejectedTotal:    rl.rejected[id],
		})
	}
	return result
}
