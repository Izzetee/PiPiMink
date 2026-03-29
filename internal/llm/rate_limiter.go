package llm

import (
	"log"
	"sync"
	"time"
)

// RateLimiter manages rate limiting for API calls to prevent
// overwhelming local LLM servers with too many requests.
type RateLimiter struct {
	lastRequestTime time.Time
	mutex           sync.Mutex
	waitSeconds     int
}

// NewRateLimiter creates a new rate limiter with the specified wait time between requests.
func NewRateLimiter(waitSeconds int) *RateLimiter {
	return &RateLimiter{
		waitSeconds: waitSeconds,
	}
}

// Wait blocks until enough time has passed since the last request.
// This prevents sending too many requests in a short period of time.
func (r *RateLimiter) Wait() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.lastRequestTime.IsZero() {
		elapsed := time.Since(r.lastRequestTime)
		waitTime := time.Duration(r.waitSeconds)*time.Second - elapsed
		if waitTime > 0 {
			log.Printf("Rate limit: Waiting %v seconds before next request", waitTime.Seconds())
			time.Sleep(waitTime)
		}
	}
}

// UpdateLastRequestTime marks the current time as the most recent request time.
// Call this after making a request to start the wait period.
func (r *RateLimiter) UpdateLastRequestTime() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.lastRequestTime = time.Now()
}
