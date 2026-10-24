package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type clubDomainTestDNS struct {
	txt   map[string][]string
	ips   map[string][]net.IPAddr
	calls int64
}

func (d *clubDomainTestDNS) LookupTXT(_ context.Context, name string) ([]string, error) {
	atomic.AddInt64(&d.calls, 1)
	return d.txt[name], nil
}
func (d *clubDomainTestDNS) LookupIPAddr(_ context.Context, name string) ([]net.IPAddr, error) {
	atomic.AddInt64(&d.calls, 1)
	return d.ips[name], nil
}

func clubDomainFixture(t *testing.T) (*_router, int64, int64, *clubDomainTestDNS) {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "domains.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	store := NewStore(db)
	rt := &_router{store: store, router: chi.NewRouter(), baseLogger: logrus.New(), sessionTimeout: time.Hour, adminSessions: map[string]adminSession{
		"master": {Role: "superadmin", ExpiresAt: time.Now().Add(time.Hour)}, "staff": {Role: "staff", ExpiresAt: time.Now().Add(time.Hour)},
	}}
	registerClubRoutes(rt)
	id, err := store.CreateClubSite(context.Background(), ClubSite{OrganizationID: 1, Slug: "aurora", Name: "Aurora", Active: true}, nil, "aurora", "hash")
	if err != nil {
		t.Fatal(err)
	}
	other, err := store.CreateClubSite(context.Background(), ClubSite{OrganizationID: 2, Slug: "other", Name: "Other", Active: true}, nil, "other", "hash")
	if err != nil {
		t.Fatal(err)
	}
	dns := &clubDomainTestDNS{txt: map[string][]string{}, ips: map[string][]net.IPAddr{}}
	rt.clubDomains.dns = dns
	rt.clubDomains.ips = []net.IP{net.ParseIP("203.0.113.10")}
	rt.clubDomains.reserved = []string{"mvp.wearingcash.it"}
	return rt, id, other, dns
}

func domainRequest(rt *_router, method, path, body, token, host string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Host = host
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	rt.router.ServeHTTP(w, r)
	return w
}

func TestClubDomainsNormalizeAndRejectUnsafeHosts(t *testing.T) {
	for raw, want := range map[string]string{" VolleyAurora.IT ": "volleyaurora.it", "https://WWW.VolleyAurora.IT/": "www.volleyaurora.it", "volley.it.": "volley.it", "xn--societ-nta.it": "xn--societ-nta.it"} {
		got, err := normalizeClubHostname(raw)
		if err != nil || got != want {
			t.Fatalf("normalize %q: %q %v", raw, got, err)
		}
	}
	for _, raw := range []string{"localhost", "127.0.0.1", "192.168.1.1", "*.volley.it", "volley.it:443", "https://user@volley.it", "https://volley.it/club/aurora", "https://volley.it/?x=1", "volley.it/path", "a..it", "-volley.it", "volley-.it", "società.it", "foo.local", "foo.test", "foo.123", strings.Repeat("a", 64) + ".it"} {
		if _, err := normalizeClubHostname(raw); err == nil {
			t.Fatalf("accepted unsafe hostname %q", raw)
		}
	}
}

func TestClubDomainsMasterAuthorizationAndAtomicUniqueness(t *testing.T) {
	rt, id, other, _ := clubDomainFixture(t)
	path := fmt.Sprintf("/admin/master/clubs/%d/domains", id)
	for token, status := range map[string]int{"": 401, "staff": 403} {
		w := domainRequest(rt, "POST", path, `{"hostname":"volley.it"}`, token, "mvp.wearingcash.it")
		if w.Code != status {
			t.Fatalf("auth %q: %d", token, w.Code)
		}
	}
	for _, host := range []string{"mvp.wearingcash.it", "x.mvp.wearingcash.it", "https://volley.it/private", "localhost"} {
		w := domainRequest(rt, "POST", path, `{"hostname":"`+host+`"}`, "master", "mvp.wearingcash.it")
		if w.Code != 400 {
			t.Fatalf("reserved/invalid: %s %d", host, w.Code)
		}
	}
	w := domainRequest(rt, "POST", path, `{"hostname":"Volley.IT","includeWWW":true}`, "master", "mvp.wearingcash.it")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	items, _ := rt.store.ListClubDomains(context.Background(), id)
	if len(items) != 2 || !items[0].Primary || items[1].Primary || items[0].VerificationValue != items[1].VerificationValue || items[0].VerificationName != "_clubhub.volley.it" {
		t.Fatalf("unexpected domain group: %+v", items)
	}
	w = domainRequest(rt, "POST", fmt.Sprintf("/admin/master/clubs/%d/domains", other), `{"hostname":"www.volley.it"}`, "master", "mvp.wearingcash.it")
	if w.Code != 409 {
		t.Fatal("duplicate accepted", w.Code)
	}
	if err := rt.store.AddClubDomains(context.Background(), other, "other.it", true); err != nil {
		t.Fatal(err)
	}
	if err := rt.store.AddClubDomains(context.Background(), id, "www.other.it", false); !errors.Is(err, errClubDomainConflict) {
		t.Fatal(err)
	}
	if err := rt.store.SetPrimaryClubDomain(context.Background(), other, items[0].ID); !errors.Is(err, errClubNotFound) {
		t.Fatal("cross-club primary accepted", err)
	}
	if err := rt.store.DeleteClubDomain(context.Background(), other, items[0].ID); !errors.Is(err, errClubNotFound) {
		t.Fatal("cross-club removal accepted", err)
	}
	// A conflict in the second insert must roll the entire group back.
	if err := rt.store.AddClubDomains(context.Background(), other, "www.fresh.it", false); err != nil {
		t.Fatal(err)
	}
	if err := rt.store.AddClubDomains(context.Background(), id, "fresh.it", true); !errors.Is(err, errClubDomainConflict) {
		t.Fatal(err)
	}
	var count int
	_ = rt.store.db.QueryRow(`SELECT COUNT(*) FROM club_domains WHERE hostname='fresh.it'`).Scan(&count)
	if count != 0 {
		t.Fatal("partial domain group saved")
	}
}

func TestClubDomainsVerificationResolutionRevocationAndExpiry(t *testing.T) {
	rt, id, _, dns := clubDomainFixture(t)
	ctx := context.Background()
	if err := rt.store.AddClubDomains(ctx, id, "volley.it", true); err != nil {
		t.Fatal(err)
	}
	items, _ := rt.store.ListClubDomains(ctx, id)
	ask := func(host string) int {
		return domainRequest(rt, "GET", "/internal/club-domains/ask?domain="+host, "", "", "backend:3000").Code
	}
	if ask("volley.it") != 403 {
		t.Fatal("pending domain can issue TLS")
	}
	dns.txt[items[0].VerificationName] = []string{"wrong-token"}
	dns.ips["volley.it"] = []net.IPAddr{{IP: net.ParseIP("203.0.113.10")}}
	rt.clubDomains.verify(ctx, items)
	if ask("volley.it") != 403 {
		t.Fatal("wrong proof accepted")
	}
	dns.txt[items[0].VerificationName] = []string{items[0].VerificationValue}
	dns.ips["www.volley.it"] = dns.ips["volley.it"]
	rt.clubDomains.verify(ctx, items)
	calls := atomic.LoadInt64(&dns.calls)
	if ask("volley.it") != 200 || ask("www.volley.it") != 200 || ask("unknown.it") != 403 {
		t.Fatal("TLS allowlist is incorrect")
	}
	if atomic.LoadInt64(&dns.calls) != calls {
		t.Fatal("TLS permission lookup performed DNS requests")
	}
	w := domainRequest(rt, "GET", "/v1/club-domain?domain=unknown.it", "", "", "www.volley.it:443")
	var resolved map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resolved)
	if w.Code != 200 || resolved["clubSlug"] != "aurora" || resolved["primaryHostname"] != "volley.it" {
		t.Fatal(w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("domain resolution may be cached")
	}
	r := httptest.NewRequest("GET", "/v1/club-domain", nil)
	r.Host = "unknown.it"
	r.Header.Set("X-Forwarded-Host", "volley.it")
	spoof := httptest.NewRecorder()
	rt.router.ServeHTTP(spoof, r)
	if spoof.Code != 204 {
		t.Fatal("forwarded-host spoof trusted")
	}
	// A cached certificate must not allow a revoked/deleted domain to serve traffic.
	if err := rt.store.DeleteClubDomain(ctx, id, items[0].ID); err != nil {
		t.Fatal(err)
	}
	if ask("volley.it") != 403 || domainRequest(rt, "GET", "/internal/club-domains/host", "", "", "volley.it").Code != 404 {
		t.Fatal("deleted domain still authorized")
	}
	_, primary, err := rt.store.ResolveClubDomain(ctx, "www.volley.it")
	if err != nil || primary != "www.volley.it" {
		t.Fatal("remaining alias not promoted", primary, err)
	}
	_, _ = rt.store.db.Exec(`UPDATE club_domains SET verified_at=?`, time.Now().Add(-25*time.Hour).UTC().Format(time.RFC3339Nano))
	if ask("www.volley.it") != 403 {
		t.Fatal("stale verification accepted")
	}
	rt.clubDomains.verify(ctx, items[1:])
	dns.ips["www.volley.it"] = append(dns.ips["www.volley.it"], net.IPAddr{IP: net.ParseIP("2001:db8::2")})
	rt.clubDomains.verify(ctx, items[1:])
	if ask("www.volley.it") != 403 {
		t.Fatal("incorrect AAAA record accepted")
	}
	dns.ips["www.volley.it"] = []net.IPAddr{{IP: net.ParseIP("203.0.113.10")}}
	rt.clubDomains.verify(ctx, items[1:])
	_, _ = rt.store.db.Exec(`UPDATE club_sites SET active=0 WHERE id=?`, id)
	if ask("www.volley.it") != 403 {
		t.Fatal("inactive club authorized")
	}
	if err := rt.store.DeleteClubSite(ctx, id); err != nil {
		t.Fatal(err)
	}
	left, _ := rt.store.ListClubDomains(ctx, id)
	if len(left) != 0 {
		t.Fatal("orphaned domain registrations")
	}
}

func TestClubDomainsAutomaticWorkerActivatesPendingDomain(t *testing.T) {
	rt, id, _, dns := clubDomainFixture(t)
	ctx := context.Background()
	if err := rt.store.AddClubDomains(ctx, id, "automatic.it", false); err != nil {
		t.Fatal(err)
	}
	items, _ := rt.store.ListClubDomains(ctx, id)
	dns.txt[items[0].VerificationName] = []string{items[0].VerificationValue}
	dns.ips["automatic.it"] = []net.IPAddr{{IP: net.ParseIP("203.0.113.10")}}
	rt.clubDomains.start()
	t.Cleanup(func() { rt.Close() })
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			t.Fatal("pending domain was not automatically activated")
		case <-ticker.C:
			if _, _, err := rt.store.ResolveClubDomain(ctx, "automatic.it"); err == nil {
				return
			}
		}
	}
}

func TestClubDomainsManualVerificationQueuesWithoutDNSInHTTPRequest(t *testing.T) {
	rt, id, _, dns := clubDomainFixture(t)
	path := fmt.Sprintf("/admin/master/clubs/%d/domains/verify", id)
	w := domainRequest(rt, "POST", path, "", "master", "mvp.wearingcash.it")
	if w.Code != 200 || atomic.LoadInt64(&dns.calls) != 0 {
		t.Fatal("verification request must not wait for DNS", w.Code, w.Body.String())
	}
	select {
	case queued := <-rt.clubDomains.wake:
		if queued != id {
			t.Fatal("wrong club queued", queued)
		}
	default:
		t.Fatal("verification was not queued")
	}
}
