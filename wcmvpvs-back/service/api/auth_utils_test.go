package api

import (
	"testing"
	"time"
)

func TestNormalizeE164(t *testing.T) {
	tests := []struct {
		in   string
		ok   bool
		want string
	}{
		{in: "+393331234567", ok: true, want: "+393331234567"},
		{in: " +393331234567 ", ok: true, want: "+393331234567"},
		{in: "393331234567", ok: false},
		{in: "+39-3331234567", ok: false},
		{in: "+12", ok: false},
	}

	for _, tt := range tests {
		got, ok := normalizeE164(tt.in)
		if ok != tt.ok {
			t.Fatalf("normalizeE164(%q) ok=%v want %v", tt.in, ok, tt.ok)
		}
		if got != tt.want {
			t.Fatalf("normalizeE164(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestAllowRate(t *testing.T) {
	bucket := map[string][]time.Time{}
	now := time.Now()
	for i := 0; i < 3; i++ {
		if !allowRate(bucket, "+393331234567", 3, 10*time.Minute, now) {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if allowRate(bucket, "+393331234567", 3, 10*time.Minute, now) {
		t.Fatal("4th request should be rate limited")
	}
	if !allowRate(bucket, "+393331234567", 3, 10*time.Minute, now.Add(11*time.Minute)) {
		t.Fatal("request after window should be allowed")
	}
}
