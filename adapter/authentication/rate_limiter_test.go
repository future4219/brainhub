package authentication

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsFiveAttemptsPerFifteenMinutes(t *testing.T) {
	limiter := NewRateLimiter(5, 15*time.Minute)
	now := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	for attempt := 1; attempt <= 5; attempt++ {
		if !limiter.Allow("ip\x00email", now) {
			t.Fatalf("attempt %d was rejected", attempt)
		}
	}
	if limiter.Allow("ip\x00email", now) {
		t.Fatal("sixth attempt was allowed")
	}
	if !limiter.Allow("ip\x00email", now.Add(15*time.Minute)) {
		t.Fatal("attempt after the window was rejected")
	}
}

func TestBcryptRejectsCostBelowTwelve(t *testing.T) {
	if _, err := NewBcrypt(11); err == nil {
		t.Fatal("bcrypt cost 11 was accepted")
	}
}
