package api

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/api/reqcontext"
)

type clubDomainDNS interface {
	LookupTXT(context.Context, string) ([]string, error)
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type clubDomainManager struct {
	store     *Store
	dns       clubDomainDNS
	target    string
	ips       []net.IP
	reserved  []string
	wake      chan int64
	cancel    context.CancelFunc
	done      chan struct{}
	startOnce sync.Once
}

func newClubDomainManager(store *Store) *clubDomainManager {
	target := strings.TrimSpace(os.Getenv("CLUB_DOMAIN_TARGET"))
	if target == "" {
		target = "mvp.wearingcash.it"
	}
	m := &clubDomainManager{store: store, dns: net.DefaultResolver, target: target, wake: make(chan int64, 16), done: make(chan struct{})}
	m.reserved = append([]string{"mvp.wearingcash.it", strings.ToLower(target)}, strings.Split(os.Getenv("CLUB_DOMAIN_RESERVED_HOSTS"), ",")...)
	for _, value := range strings.Split(os.Getenv("CLUB_DOMAIN_TARGET_IPS"), ",") {
		if ip := net.ParseIP(strings.TrimSpace(value)); ip != nil {
			m.ips = append(m.ips, ip)
		}
	}
	return m
}

func normalizeClubHostname(value string) (string, error) {
	host := strings.ToLower(strings.TrimSpace(value))
	if strings.Contains(host, "://") {
		u, err := url.Parse(host)
		if err != nil || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return "", errors.New("invalid_domain")
		}
		host = u.Host
	}
	host = strings.TrimSuffix(host, ".")
	if len(host) > 253 || net.ParseIP(host) != nil {
		return "", errors.New("invalid_domain")
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return "", errors.New("invalid_domain")
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", errors.New("invalid_domain")
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return "", errors.New("invalid_domain")
			}
		}
	}
	tld := labels[len(labels)-1]
	if len(tld) < 2 || strings.Trim(tld, "0123456789") == "" {
		return "", errors.New("invalid_domain")
	}
	for _, suffix := range []string{"localhost", "local", "internal", "invalid", "test", "example", "lan", "home"} {
		if tld == suffix {
			return "", errors.New("invalid_domain")
		}
	}
	return host, nil
}

func requestClubHostname(r *http.Request) (string, error) {
	host := r.Host
	if strings.Contains(host, ":") {
		var err error
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			return "", err
		}
	}
	// Only Host is used; client-supplied forwarded-host headers are not trusted.
	return normalizeClubHostname(host)
}

func (m *clubDomainManager) isReserved(host string) bool {
	for _, reserved := range m.reserved {
		reserved = strings.ToLower(strings.TrimSpace(reserved))
		if reserved != "" && (host == reserved || strings.HasSuffix(host, "."+reserved)) {
			return true
		}
	}
	return false
}

func containsClubIP(ips []net.IP, ip net.IP) bool {
	for _, expected := range ips {
		if expected.Equal(ip) {
			return true
		}
	}
	return false
}

func (m *clubDomainManager) check(ctx context.Context, d ClubDomain) (bool, string) {
	if m.isReserved(d.Hostname) {
		return false, "Il dominio è riservato alla piattaforma."
	}
	txts, err := m.dns.LookupTXT(ctx, d.VerificationName)
	if err != nil {
		return false, "Record TXT di verifica non ancora disponibile."
	}
	owned := false
	for _, txt := range txts {
		if txt == d.VerificationValue {
			owned = true
			break
		}
	}
	if !owned {
		return false, "Il record TXT non contiene il codice di verifica indicato."
	}
	expected := m.ips
	if len(expected) == 0 {
		addresses, err := m.dns.LookupIPAddr(ctx, m.target)
		if err != nil {
			return false, "Destinazione del server non verificabile. Riprova tra poco."
		}
		for _, address := range addresses {
			expected = append(expected, address.IP)
		}
	}
	if len(expected) == 0 {
		return false, "Destinazione del server non configurata."
	}
	actual, err := m.dns.LookupIPAddr(ctx, d.Hostname)
	if err != nil || len(actual) == 0 {
		return false, "Record A/AAAA o CNAME del dominio non ancora disponibile."
	}
	for _, address := range actual {
		ip := address.IP
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || !containsClubIP(expected, ip) {
			return false, "Il dominio deve puntare al server ClubHub: controlla tutti i record A e AAAA."
		}
	}
	return true, "Dominio verificato e autorizzato. HTTPS viene attivato automaticamente al primo accesso."
}

func (m *clubDomainManager) verify(ctx context.Context, domains []ClubDomain) {
	jobs := make(chan ClubDomain)
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for d := range jobs {
				checkCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
				valid, message := m.check(checkCtx, d)
				cancel()
				if ctx.Err() == nil {
					_ = m.store.UpdateClubDomainCheck(ctx, d, valid, message)
				}
			}
		}()
	}
	for _, d := range domains {
		select {
		case jobs <- d:
		case <-ctx.Done():
			close(jobs)
			workers.Wait()
			return
		}
	}
	close(jobs)
	workers.Wait()
}

func (m *clubDomainManager) start() {
	m.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancel = cancel
		go func() {
			defer close(m.done)
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			var siteID int64
			for {
				domains, err := m.store.ListClubDomains(ctx, siteID)
				if err == nil {
					m.verify(ctx, domains)
				}
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					siteID = 0
				case siteID = <-m.wake:
				}
			}
		}()
	})
}

func registerClubDomainRoutes(rt *_router) {
	if err := rt.store.ensureClubDomainTables(); err != nil {
		panic("club domains: " + err.Error())
	}
	if rt.clubDomains == nil {
		rt.clubDomains = newClubDomainManager(rt.store)
	}
	rt.router.Get("/admin/master/clubs/{id}/domains", rt.wrapAdmin(rt.masterClubDomains))
	rt.router.Post("/admin/master/clubs/{id}/domains", rt.wrapAdmin(rt.masterAddClubDomain))
	rt.router.Post("/admin/master/clubs/{id}/domains/verify", rt.wrapAdmin(rt.masterVerifyClubDomains))
	rt.router.Put("/admin/master/clubs/{id}/domains/{domainID}/primary", rt.wrapAdmin(rt.masterPrimaryClubDomain))
	rt.router.Delete("/admin/master/clubs/{id}/domains/{domainID}", rt.wrapAdmin(rt.masterDeleteClubDomain))
	rt.router.Get("/v1/club-domain", rt.publicClubDomain)
	// Reachable only inside the Docker network; Caddy blocks public /internal routes.
	rt.router.Get("/internal/club-domains/ask", rt.allowClubDomainTLS)
	rt.router.Get("/internal/club-domains/host", rt.allowClubDomainHost)
}

func (rt *_router) domainResponse(w http.ResponseWriter, r *http.Request, siteID int64) {
	domains, err := rt.store.ListClubDomains(r.Context(), siteID)
	if err != nil {
		http.Error(w, `{"error":"internal"}`, 500)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	ips := []string{}
	for _, ip := range rt.clubDomains.ips {
		ips = append(ips, ip.String())
	}
	writeJSON(w, 200, map[string]interface{}{"domains": domains, "target": rt.clubDomains.target, "targetIPs": ips, "automaticCheckSeconds": 60})
}

func (rt *_router) domainSiteID(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) (int64, bool) {
	if !rt.ensureSuperAdmin(w, ctx) {
		return 0, false
	}
	id, err := parseClubID(r, "id")
	if err != nil || id <= 0 {
		http.Error(w, `{"error":"bad_id"}`, 400)
		return 0, false
	}
	if _, err = rt.store.ClubSiteByID(r.Context(), id); err != nil {
		http.Error(w, `{"error":"club_not_found"}`, 404)
		return 0, false
	}
	return id, true
}

func (rt *_router) masterClubDomains(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, ok := rt.domainSiteID(w, r, ctx)
	if ok {
		rt.domainResponse(w, r, id)
	}
}

func (rt *_router) masterAddClubDomain(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, ok := rt.domainSiteID(w, r, ctx)
	if !ok {
		return
	}
	var input struct {
		Hostname   string `json:"hostname"`
		IncludeWWW bool   `json:"includeWWW"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input) != nil {
		http.Error(w, `{"error":"bad_json"}`, 400)
		return
	}
	host, err := normalizeClubHostname(input.Hostname)
	if err != nil || rt.clubDomains.isReserved(host) {
		http.Error(w, `{"error":"invalid_domain"}`, 400)
		return
	}
	includeWWW := input.IncludeWWW && !strings.HasPrefix(host, "www.")
	if includeWWW {
		if _, err = normalizeClubHostname("www." + host); err != nil {
			http.Error(w, `{"error":"invalid_domain"}`, 400)
			return
		}
	}
	err = rt.store.AddClubDomains(r.Context(), id, host, includeWWW)
	if errors.Is(err, errClubDomainConflict) {
		http.Error(w, `{"error":"domain_taken"}`, 409)
		return
	}
	if err != nil {
		if err.Error() == "domain_limit" {
			http.Error(w, `{"error":"domain_limit"}`, 400)
		} else {
			http.Error(w, `{"error":"save_failed"}`, 500)
		}
		return
	}
	select {
	case rt.clubDomains.wake <- id:
	default:
	}
	rt.domainResponse(w, r, id)
}

func (rt *_router) masterVerifyClubDomains(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, ok := rt.domainSiteID(w, r, ctx)
	if !ok {
		return
	}
	// DNS I/O runs in the worker: the HTTP server has a short write timeout.
	select {
	case rt.clubDomains.wake <- id:
	default:
	}
	rt.domainResponse(w, r, id)
}

func (rt *_router) masterPrimaryClubDomain(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, ok := rt.domainSiteID(w, r, ctx)
	if !ok {
		return
	}
	domainID, err := parseClubID(r, "domainID")
	if err == nil {
		err = rt.store.SetPrimaryClubDomain(r.Context(), id, domainID)
	}
	if err != nil {
		http.Error(w, `{"error":"domain_not_found"}`, 404)
		return
	}
	rt.domainResponse(w, r, id)
}

func (rt *_router) masterDeleteClubDomain(w http.ResponseWriter, r *http.Request, ctx reqcontext.RequestContext) {
	id, ok := rt.domainSiteID(w, r, ctx)
	if !ok {
		return
	}
	domainID, err := parseClubID(r, "domainID")
	if err == nil {
		err = rt.store.DeleteClubDomain(r.Context(), id, domainID)
	}
	if err != nil {
		http.Error(w, `{"error":"domain_not_found"}`, 404)
		return
	}
	rt.domainResponse(w, r, id)
}

func (rt *_router) publicClubDomain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	host, err := requestClubHostname(r)
	if err != nil || rt.clubDomains.isReserved(host) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	slug, primary, err := rt.store.ResolveClubDomain(r.Context(), host)
	if clubDomainNotFound(err) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"domain_unavailable"}`, 503)
		return
	}
	writeJSON(w, 200, map[string]string{"clubSlug": slug, "primaryHostname": primary})
}

func (rt *_router) allowClubDomainTLS(w http.ResponseWriter, r *http.Request) {
	host, err := normalizeClubHostname(r.URL.Query().Get("domain"))
	if err == nil && !rt.clubDomains.isReserved(host) {
		_, _, err = rt.store.ResolveClubDomain(r.Context(), host)
		if err == nil {
			w.WriteHeader(200)
			return
		}
	}
	w.WriteHeader(http.StatusForbidden)
}

func (rt *_router) allowClubDomainHost(w http.ResponseWriter, r *http.Request) {
	host, err := requestClubHostname(r)
	if err == nil && !rt.clubDomains.isReserved(host) {
		_, _, err = rt.store.ResolveClubDomain(r.Context(), host)
		if err == nil {
			w.WriteHeader(200)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}
