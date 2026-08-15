package authentication

import (
	"sync"
	"time"
)

type attemptWindow struct {
	startedAt time.Time
	attempts  int
}

type RateLimiter struct {
	mu          sync.Mutex
	limit       int
	window      time.Duration
	lastCleanup time.Time
	attempts    map[string]attemptWindow
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, attempts: make(map[string]attemptWindow)}
}

func (l *RateLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// ponytail: this limiter is process-local; use a shared store when the API runs on multiple replicas.
	if l.lastCleanup.IsZero() || now.Sub(l.lastCleanup) >= l.window {
		for key, attempt := range l.attempts {
			if now.Sub(attempt.startedAt) >= l.window {
				delete(l.attempts, key)
			}
		}
		l.lastCleanup = now
	}

	attempt := l.attempts[key]
	if attempt.startedAt.IsZero() || now.Sub(attempt.startedAt) >= l.window {
		attempt = attemptWindow{startedAt: now}
	}
	if attempt.attempts >= l.limit {
		return false
	}
	attempt.attempts++
	l.attempts[key] = attempt
	return true
}

func (l *RateLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}
