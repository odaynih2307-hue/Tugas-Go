package helper

import (
	"sync"
	"time"
)

// LoginRateLimiter melacak percobaan gagal login dalam sliding window 1 menit.
type LoginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

var GlobalLoginLimiter = NewLoginRateLimiter(5, 1*time.Minute)

// NewLoginRateLimiter membuat rate limiter baru.
func NewLoginRateLimiter(limit int, window time.Duration) *LoginRateLimiter {
	return &LoginRateLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// RecordFailure mencatat 1 kegagalan login dan mengembalikan apakah limit telah terlampaui.
func (rl *LoginRateLimiter) RecordFailure(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Filter timestamps lama
	var validTimes []time.Time
	for _, t := range rl.attempts[key] {
		if t.After(cutoff) {
			validTimes = append(validTimes, t)
		}
	}

	validTimes = append(validTimes, now)
	rl.attempts[key] = validTimes

	return len(validTimes) > rl.limit
}

// IsRateLimited memeriksa apakah key saat ini melebihi ambang batas kegagalan.
func (rl *LoginRateLimiter) IsRateLimited(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	var validTimes []time.Time
	for _, t := range rl.attempts[key] {
		if t.After(cutoff) {
			validTimes = append(validTimes, t)
		}
	}
	rl.attempts[key] = validTimes

	return len(validTimes) > rl.limit
}

// Reset menghapus riwayat kegagalan (misalnya saat login berhasil).
func (rl *LoginRateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, key)
}
