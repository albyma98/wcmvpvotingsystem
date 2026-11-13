package api

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/sirupsen/logrus"
)

const matchdayGrantHeader = "X-Matchday-Grant"

type matchdayRequestKey struct {
	organization, event, status int
	route                       string
	latency, started            int64
}

type matchdayRequestRecorder struct {
	db       *sql.DB
	logger   logrus.FieldLogger
	secret   string
	mu       sync.Mutex
	pending  map[matchdayRequestKey]int
	shutdown chan struct{}
	done     chan struct{}
	stopOnce sync.Once
	stopped  bool
}

func (rt *_router) newMatchdayRequestRecorder() (*matchdayRequestRecorder, error) {
	secret := rt.VoteSecret
	if !rt.sponsorAnalyticsConfigured() {
		var seed [32]byte
		if _, err := rand.Read(seed[:]); err != nil {
			return nil, err
		}
		secret = base64.RawURLEncoding.EncodeToString(seed[:])
	}
	m := &matchdayRequestRecorder{
		db: rt.db.SQLConn(), logger: rt.baseLogger, secret: secret, pending: make(map[matchdayRequestKey]int),
		shutdown: make(chan struct{}), done: make(chan struct{}),
	}
	go m.run()
	return m, nil
}

func (m *matchdayRequestRecorder) grant(organization, event int, now time.Time) string {
	payload := fmt.Sprintf("%d:%d:%d", organization, event, now.Add(24*time.Hour).Unix())
	return payload + "." + base64.RawURLEncoding.EncodeToString(sponsorMAC(m.secret, "matchday-request", payload))
}

func (m *matchdayRequestRecorder) grantEvent(grant string, organization int, now time.Time) int {
	if len(grant) > 160 {
		return 0
	}
	payload, signature, ok := strings.Cut(grant, ".")
	parts := strings.Split(payload, ":")
	if !ok || len(parts) != 3 || parts[0] != strconv.Itoa(organization) {
		return 0
	}
	event, eventErr := strconv.Atoi(parts[1])
	expires, expiryErr := strconv.ParseInt(parts[2], 10, 64)
	mac, signatureErr := base64.RawURLEncoding.DecodeString(signature)
	if eventErr != nil || event <= 0 || expiryErr != nil || expires <= now.Unix() ||
		signatureErr != nil || !hmac.Equal(mac, sponsorMAC(m.secret, "matchday-request", payload)) {
		return 0
	}
	return event
}

func matchdayObservedRoute(r *http.Request, event int) string {
	path := r.URL.Path
	if r.Method == http.MethodOptions || strings.Contains(r.Header.Get("Accept"), "text/event-stream") ||
		strings.HasSuffix(path, "/stream") || strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return ""
	}
	if eventText := r.URL.Query().Get("event_id"); eventText != "" && eventText != strconv.Itoa(event) {
		return ""
	}
	if slug, merchPath, ok := strings.Cut(strings.TrimPrefix(path, "/v1/clubs/"), "/"); strings.HasPrefix(path, "/v1/clubs/") && ok && slug != "" {
		if matchID := r.URL.Query().Get("match_id"); matchID != "" && matchID != strconv.Itoa(event) {
			return ""
		}
		switch merchPath {
		case "merch/events", "merch/orders":
			if r.Method == http.MethodPost {
				return "merch_attributed"
			}
			return "merch"
		case "merch", "merch/quote":
			return "merch"
		default:
			if strings.HasPrefix(merchPath, "merch/orders/") {
				return "merch"
			}
		}
	}
	switch path {
	case "/vote":
		return "vote"
	case "/active-event", "/organization-theme", "/public/players", "/sponsors":
		return "home"
	}
	switch {
	case strings.HasPrefix(path, "/bar/") && !strings.Contains(path, "webhook"):
		return "bar"
	case strings.HasPrefix(path, "/fan/"), strings.HasPrefix(path, "/auth/"), strings.HasPrefix(path, "/coupons/"):
		return "fan"
	}
	for _, prefix := range []string{"/events/", "/public/events/"} {
		if rest := strings.TrimPrefix(path, prefix); rest != path {
			id, operation, ok := strings.Cut(rest, "/")
			if !ok || id != strconv.Itoa(event) || operation == "report-signals" || operation == "tracking/events" {
				return ""
			}
			return "event"
		}
	}
	return ""
}

type matchdayRequestContextKey struct{}
type matchdayRequestContext struct {
	validatedEvent int
}

func validateMatchdayRequestEvent(r *http.Request, event int) {
	if observation, ok := r.Context().Value(matchdayRequestContextKey{}).(*matchdayRequestContext); ok {
		observation.validatedEvent = event
	}
}

func (rt *_router) observeMatchdayRequest(w http.ResponseWriter, r *http.Request, organization int, started time.Time, serve func(http.ResponseWriter, *http.Request)) {
	m := rt.matchdayRequests
	if m == nil || organization <= 0 {
		serve(w, r)
		return
	}
	initial := r.URL.Path == "/active-event" && r.URL.Query().Get("matchday") == "1"
	event := m.grantEvent(r.Header.Get(matchdayGrantHeader), organization, started)
	if event == 0 && !initial {
		serve(w, r)
		return
	}
	route := matchdayObservedRoute(r, event)
	if route == "" {
		serve(w, r)
		return
	}
	observation := &matchdayRequestContext{}
	r = r.WithContext(context.WithValue(r.Context(), matchdayRequestContextKey{}, observation))
	metrics := httpsnoop.CaptureMetricsFn(w, func(w http.ResponseWriter) { serve(w, r) })
	if initial && observation.validatedEvent > 0 {
		event = observation.validatedEvent
	}
	if (route == "vote" || route == "merch_attributed") && event != observation.validatedEvent {
		return
	}
	if route == "merch_attributed" {
		route = "merch"
	}
	if event > 0 {
		m.record(matchdayRequestKey{
			organization: organization, event: event, status: metrics.Code, route: route,
			latency: time.Since(started).Milliseconds(), started: started.UnixMilli(),
		})
	}
}

func (m *matchdayRequestRecorder) record(key matchdayRequestKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.stopped {
		m.pending[key]++
	}
}

func (m *matchdayRequestRecorder) run() {
	defer close(m.done)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.flush()
		case <-m.shutdown:
			m.flush()
			return
		}
	}
}

func (m *matchdayRequestRecorder) close() {
	m.stopOnce.Do(func() {
		m.mu.Lock()
		m.stopped = true
		m.mu.Unlock()
		close(m.shutdown)
	})
	<-m.done
}

func (m *matchdayRequestRecorder) flush() {
	m.mu.Lock()
	pending := m.pending
	m.pending = make(map[matchdayRequestKey]int)
	m.mu.Unlock()
	if len(pending) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.save(ctx, pending); err != nil {
		m.logger.WithError(err).Warn("could not store matchday API observations")
		m.mu.Lock()
		for key, count := range pending {
			m.pending[key] += count
		}
		m.mu.Unlock()
	}
}

func (m *matchdayRequestRecorder) save(ctx context.Context, observations map[matchdayRequestKey]int) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	statement, err := tx.PrepareContext(ctx, `INSERT INTO matchday_api_requests (event_id,organization_id,route,http_status,latency_ms,samples)
		SELECT e.id,e.organization_id,?,?,?,? FROM events e
		WHERE e.id=? AND e.organization_id=? AND EXISTS (
			SELECT 1 FROM matchday_technology_windows w WHERE w.event_id=e.id AND w.organization_id=e.organization_id
				AND w.opened_at<=? AND (w.closed_at IS NULL OR w.closed_at>?)
		)
		ON CONFLICT(event_id,route,http_status,latency_ms) DO UPDATE SET samples=samples+excluded.samples`)
	if err != nil {
		return err
	}
	defer statement.Close()
	for key, count := range observations {
		if _, err := statement.ExecContext(ctx, key.route, key.status, key.latency, count, key.event, key.organization, key.started, key.started); err != nil {
			return err
		}
	}
	return tx.Commit()
}
