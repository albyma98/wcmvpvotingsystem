package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestMatchdayManagedQRCountsOpensAndDeduplicatesArrivals(t *testing.T) {
	f := newEngagementReportFixture(t)
	if _, err := f.conn.Exec(`UPDATE events SET is_active=1 WHERE id=?`, f.event); err != nil {
		t.Fatal(err)
	}
	target := "/" + f.org.Slug + "/new?utm_source=led&utm_medium=qr"
	if _, err := f.router.db.UpsertQRRedirect("/stadium-qr", target); err != nil {
		t.Fatal(err)
	}
	open := func(method, purpose string) string {
		t.Helper()
		r := httptest.NewRequest(method, "https://example.com/stadium-qr", nil)
		r.Header.Set("Sec-Purpose", purpose)
		w := httptest.NewRecorder()
		f.router.handleQRRedirectNotFound(w, r, f.ctx)
		engagementReportRequireStatus(t, w, http.StatusFound)
		location, err := url.Parse(w.Header().Get("Location"))
		if err != nil || location.Query().Get("utm_source") != "led" {
			t.Fatalf("unexpected redirect %s, %v", location, err)
		}
		return location.Query().Get(matchdayQRParameter)
	}
	if token := open(http.MethodHead, ""); token != "" {
		t.Fatal("HEAD generated a QR arrival")
	}
	if token := open(http.MethodGet, "prefetch"); token != "" {
		t.Fatal("prefetch generated a QR arrival")
	}
	first, second := open(http.MethodGet, ""), open(http.MethodGet, "")
	if first == second || matchdayQRTokenKey(first) == "" || matchdayQRTokenKey(second) == "" {
		t.Fatal("expected unique, opaque QR arrival tokens")
	}
	send := func(id, token, device string) {
		t.Helper()
		view := engagementReportSignalPayload(id, "page_view", map[string]interface{}{"entry_channel": "direct"})
		view["qr_arrival"] = token
		w := engagementReportRequest(t, f.router.recordReportSignals, http.MethodPost, f.event,
			map[string]interface{}{"events": []interface{}{view}}, f.ctx, map[string]string{"X-Device-Id": device})
		engagementReportRequireStatus(t, w, http.StatusNoContent)
	}
	send("first-view", first, "browser-a")
	send("first-view", first, "browser-a")
	send("first-view", second, "browser-a")
	send("forwarded-view", first, "browser-b")
	var firstChannel, forwardedChannel string
	if err := f.conn.QueryRow(`SELECT json_extract(metadata,'$.entry_channel') FROM engagement_report_signals WHERE event_id=? AND signal_id='first-view'`, f.event).Scan(&firstChannel); err != nil || firstChannel != "led" {
		t.Fatalf("channel=%q err=%v", firstChannel, err)
	}
	if err := f.conn.QueryRow(`SELECT json_extract(metadata,'$.entry_channel') FROM engagement_report_signals WHERE event_id=? AND signal_id='forwarded-view'`, f.event).Scan(&forwardedChannel); err != nil || forwardedChannel != "direct" {
		t.Fatalf("forwarded channel=%q err=%v", forwardedChannel, err)
	}
	facts := &matchdayFacts{Metrics: map[string]matchdayMetric{}, Resolve: func(matchdayIdentity) string { return "" }}
	if err := loadMatchdayQRArrivals(context.Background(), f.conn, f.org.ID, map[int]*matchdayFacts{f.event: facts}); err != nil {
		t.Fatal(err)
	}
	calculateMatchdayMarketing(facts, nil)
	metric := facts.Metrics["marketing_qr_conversion"]
	matchdayRequireValue(t, metric.Value, 50)
	if metric.Source != "db" || metric.Status != "available" || len(metric.Missing) != 0 {
		t.Fatalf("unexpected automatic KPI %+v", metric)
	}
}
