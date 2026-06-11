package RateLimiter

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu sync.Mutex

	limits map[string]*APILimit
}

type APILimit struct {
	Total     int
	Remaining int
	ResetsAt  time.Time
}

func New() *RateLimiter {
	return &RateLimiter{
		limits: make(map[string]*APILimit),
	}
}

func (l *RateLimiter) Update(apiName string, total, remaining int, resetSeconds int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.limits[apiName] = &APILimit{
		Total:     total,
		Remaining: remaining,
		ResetsAt:  time.Now().Add(time.Duration(resetSeconds) * time.Second),
	}
}

func (l *RateLimiter) Allow(apiName string) (allowed bool, waitSeconds int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	limit, exists := l.limits[apiName]
	if !exists {
		return true, 0
	}

	if time.Now().After(limit.ResetsAt) {
		return true, 0
	}

	if limit.Remaining > 0 {
		return true, 0
	}

	waitSeconds = int(time.Until(limit.ResetsAt).Seconds())
	if waitSeconds < 0 {
		waitSeconds = 0
	}

	return false, waitSeconds
}
