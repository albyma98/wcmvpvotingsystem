package api

import (
	"strings"
	"time"
	"unicode"
)

func normalizeE164(input string) (string, bool) {
	phone := strings.TrimSpace(input)
	if len(phone) < 8 || len(phone) > 16 {
		return "", false
	}
	if !strings.HasPrefix(phone, "+") {
		return "", false
	}
	for _, r := range phone[1:] {
		if !unicode.IsDigit(r) {
			return "", false
		}
	}
	return phone, true
}

func allowRate(bucket map[string][]time.Time, key string, max int, window time.Duration, now time.Time) bool {
	if key == "" {
		return true
	}
	cutoff := now.Add(-window)
	prev := bucket[key]
	filtered := prev[:0]
	for _, ts := range prev {
		if ts.After(cutoff) {
			filtered = append(filtered, ts)
		}
	}
	if len(filtered) >= max {
		bucket[key] = filtered
		return false
	}
	bucket[key] = append(filtered, now)
	return true
}
