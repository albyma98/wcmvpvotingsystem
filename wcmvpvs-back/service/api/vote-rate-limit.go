package api

import (
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	voteDeviceLimit       = 5
	voteDeviceWindow      = 30 * time.Second
	voteIPLimit           = 40
	voteIPWindow          = 10 * time.Second
	voteThrottleMessage   = "Stai tentando di votare troppo frequentemente. Attendi qualche secondo e riprova."
	voteThrottleIPMessage = "Troppi tentativi ravvicinati da questo indirizzo. Attendi qualche istante e riprova."
)

var voteThrottleMessages = map[string]string{
	"device": voteThrottleMessage,
	"ip":     voteThrottleIPMessage,
}

func (rt *_router) getClientIP(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func pruneAttempts(attempts []time.Time, now time.Time, window time.Duration) []time.Time {
	if len(attempts) == 0 {
		return attempts
	}
	keep := attempts[:0]
	for _, attempt := range attempts {
		if now.Sub(attempt) <= window {
			keep = append(keep, attempt)
		}
	}
	return keep
}

func (rt *_router) recordAttempt(store map[string][]time.Time, key string, now time.Time, limit int, window time.Duration) bool {
	attempts := pruneAttempts(store[key], now, window)
	if len(attempts) >= limit {
		store[key] = attempts
		return false
	}
	attempts = append(attempts, now)
	if len(attempts) == 0 {
		delete(store, key)
	} else {
		store[key] = attempts
	}
	return true
}

func (rt *_router) shouldThrottleVoteAttempt(deviceID, ip string, now time.Time) (bool, string) {
	rt.voteRateMu.Lock()
	defer rt.voteRateMu.Unlock()

	if deviceID != "" {
		if !rt.recordAttempt(rt.voteRateByDevice, deviceID, now, voteDeviceLimit, voteDeviceWindow) {
			return true, voteThrottleMessages["device"]
		}
	}

	if ip != "" {
		if !rt.recordAttempt(rt.voteRateByIP, ip, now, voteIPLimit, voteIPWindow) {
			return true, voteThrottleMessages["ip"]
		}
	}

	return false, ""
}

// helper functions defined in this file are used by vote handlers to keep track of attempts
