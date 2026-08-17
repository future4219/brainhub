package output_port

import "time"

type RateLimiter interface {
	Allow(key string, now time.Time) bool
	Reset(key string)
}
