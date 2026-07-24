package api

// ============================================================================
// STORE SQLITE + CACHE — layer dati per la modalità torneo.
// Modello: un torneo è un `event` con type='tournament'. NON una tabella
// separata → riusa lo stesso schema eventi dell'app club (vedi la migration
// additiva ensureTournamentTables in service/database/database.go).
// ============================================================================

import (
	"context"
	"database/sql"
	"sync"
	"time"
)

// --- Cache TTL in-memory (per il polling /live) ------------------------------

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

type TTLCache struct {
	mu   sync.RWMutex
	data map[string]cacheEntry
}

func NewTTLCache() *TTLCache {
	c := &TTLCache{data: make(map[string]cacheEntry)}
	// GC leggero ogni 30s: un torneo ha pochi slug attivi, la mappa resta minuscola
	go func() {
		t := time.NewTicker(30 * time.Second)
		for range t.C {
			now := time.Now()
			c.mu.Lock()
			for k, e := range c.data {
				if now.After(e.expiresAt) {
					delete(c.data, k)
				}
			}
			c.mu.Unlock()
		}
	}()
	return c
}

func (c *TTLCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.data[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

func (c *TTLCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = cacheEntry{value: value, expiresAt: time.Now().Add(ttl)}
}

// --- Store -------------------------------------------------------------------

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// GetTournamentHome assembla lo snapshot completo.
func (s *Store) GetTournamentHome(ctx context.Context, slug string) (*HomeResponse, error) {
	info, err := s.getTournamentInfo(ctx, slug)
	if err != nil {
		return nil, err
	}
	live, err := s.getLiveMatches(ctx, info.Slug)
	if err != nil {
		return nil, err
	}
	next, _ := s.getNextMatch(ctx, info.Slug) // next può mancare (torneo finito): non è errore
	tiles, err := s.getTiles(ctx, info.Slug)
	if err != nil {
		return nil, err
	}
	sponsors, err := s.getSponsors(ctx, info.Slug)
	if err != nil {
		return nil, err
	}
	shopProducts, err := s.getShopProducts(ctx, info.Slug)
	if err != nil {
		return nil, err
	}
	return &HomeResponse{
		Tournament:   *info,
		LiveMatches:  live,
		NextMatch:    next,
		Tiles:        tiles,
		Sponsors:     sponsors,
		ShopProducts: shopProducts,
	}, nil
}

// GetTournamentLive è il payload leggero del polling: solo ciò che cambia
// durante il gioco (score, set, prossima partita, fase).
func (s *Store) GetTournamentLive(ctx context.Context, slug string) (*LiveResponse, error) {
	info, err := s.getTournamentInfo(ctx, slug)
	if err != nil {
		return nil, err
	}
	live, err := s.getLiveMatches(ctx, slug)
	if err != nil {
		return nil, err
	}
	next, _ := s.getNextMatch(ctx, slug)
	return &LiveResponse{
		Tournament:  &TournamentPhase{StatusLabel: info.StatusLabel, PhaseLabel: info.PhaseLabel},
		LiveMatches: live,
		NextMatch:   next,
	}, nil
}

// --- Query private (adattate allo schema reale ArenaBoostX) ------------------

func (s *Store) getTournamentInfo(ctx context.Context, slug string) (*TournamentInfo, error) {
	const q = `
		SELECT slug, COALESCE(name,''), COALESCE(format,''), COALESCE(date_label,''),
		       COALESCE(location,''), COALESCE(status_label,''), COALESCE(phase_label,''),
		       COALESCE(logo_url,''), COALESCE(hero_image_url,''), COALESCE(fan_layout,'classic'),
		       COALESCE(prizes_json,'')
		FROM events
		WHERE slug = ? AND type = 'tournament'
		LIMIT 1`
	var t TournamentInfo
	var prizesJSON string
	err := s.db.QueryRowContext(ctx, q, slug).Scan(
		&t.Slug, &t.Name, &t.Format, &t.DateLabel, &t.Location,
		&t.StatusLabel, &t.PhaseLabel, &t.Logo, &t.HeroImage, &t.Layout, &prizesJSON,
	)
	t.Prizes = decodeTournamentPrizes(prizesJSON)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) getShopProducts(ctx context.Context, slug string) ([]TournamentShopProduct, error) {
	const q = `
		SELECT p.id, p.image_url, p.title, p.description, p.price_cents, p.extras_json
		FROM tournament_shop_products p
		JOIN events e ON e.id = p.event_id AND e.slug = ? AND e.type = 'tournament'
		WHERE p.active = 1
		ORDER BY p.position, p.id`
	rows, err := s.db.QueryContext(ctx, q, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TournamentShopProduct, 0, 8)
	for rows.Next() {
		var product TournamentShopProduct
		var extrasJSON string
		if err := rows.Scan(&product.ID, &product.ImageURL, &product.Title, &product.Description,
			&product.PriceCents, &extrasJSON); err != nil {
			return nil, err
		}
		product.Extras = decodeTournamentShopExtras(extrasJSON)
		out = append(out, product)
	}
	return out, rows.Err()
}

func (s *Store) getLiveMatches(ctx context.Context, slug string) ([]LiveMatch, error) {
	const q = `
		SELECT m.id, m.court, m.set_label, m.score_a, m.score_b, m.cur_a, m.cur_b, m.sets_json,
		       ta.name, COALESCE(ta.logo_url,''),
		       tb.name, COALESCE(tb.logo_url,'')
		FROM matches m
		JOIN events e   ON e.id = m.event_id AND e.slug = ? AND e.type = 'tournament'
		JOIN tournament_teams ta ON ta.id = m.team_a_id
		JOIN tournament_teams tb ON tb.id = m.team_b_id
		WHERE m.status = 'live'
		ORDER BY m.court`
	rows, err := s.db.QueryContext(ctx, q, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]LiveMatch, 0, 4)
	for rows.Next() {
		var m LiveMatch
		var setsJSON string
		if err := rows.Scan(&m.ID, &m.Court, &m.SetLabel, &m.Score.A, &m.Score.B, &m.Cur.A, &m.Cur.B, &setsJSON,
			&m.TeamA.Name, &m.TeamA.Logo, &m.TeamB.Name, &m.TeamB.Logo); err != nil {
			return nil, err
		}
		m.Sets = decodeSets(setsJSON) // ["21-18","18-21"]
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) getNextMatch(ctx context.Context, slug string) (*NextMatch, error) {
	const q = `
		SELECT m.court, m.scheduled_time,
		       ta.name, COALESCE(ta.city,''),
		       tb.name, COALESCE(tb.city,'')
		FROM matches m
		JOIN events e  ON e.id = m.event_id AND e.slug = ? AND e.type = 'tournament'
		JOIN tournament_teams ta ON ta.id = m.team_a_id
		JOIN tournament_teams tb ON tb.id = m.team_b_id
		WHERE m.status = 'scheduled'
		ORDER BY
		  CASE WHEN m.scheduled_time >= COALESCE(
		         (SELECT a.scheduled_time FROM matches a
		            WHERE a.event_id = m.event_id AND a.is_anchor = 1 LIMIT 1), '')
		       THEN 0 ELSE 1 END,
		  m.scheduled_time, m.court
		LIMIT 1`
	var n NextMatch
	err := s.db.QueryRowContext(ctx, q, slug).Scan(
		&n.Court, &n.Time, &n.TeamA.Name, &n.TeamA.Sub, &n.TeamB.Name, &n.TeamB.Sub,
	)
	if err != nil {
		return nil, err // sql.ErrNoRows gestito dal chiamante come "nessuna prossima"
	}
	return &n, nil
}

func (s *Store) getTiles(ctx context.Context, slug string) ([]Tile, error) {
	const q = `
		SELECT t.id, t.icon, t.label, t.sub, t.color, t.route
		FROM event_tiles t
		JOIN events e ON e.id = t.event_id AND e.slug = ? AND e.type = 'tournament'
		WHERE t.enabled = 1
		ORDER BY t.position`
	rows, err := s.db.QueryContext(ctx, q, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Tile, 0, 8)
	for rows.Next() {
		var t Tile
		if err := rows.Scan(&t.ID, &t.Icon, &t.Label, &t.Sub, &t.Color, &t.Route); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// getSponsors — adattato allo schema reale: la tabella `sponsors` è scoped per
// organizzazione (non per evento) e usa colonne diverse (logo_data, link_url,
// is_active, position). Uniamo gli sponsor dell'organizzazione che possiede il
// torneo e sintetizziamo il tier dalla posizione (slot 1 = main). Così gli
// sponsor esistenti dell'app club vengono riusati senza duplicare tabelle.
func (s *Store) getSponsors(ctx context.Context, slug string) ([]Sponsor, error) {
	// Tabella dedicata del mondo torneo (NON la sponsors del club, che è
	// scopata su organization con logo_data/position: schema incompatibile).
	const q = `
		SELECT sp.id, sp.name, COALESCE(sp.logo_url,''), COALESCE(sp.url,''),
		       sp.tier, COALESCE(sp.brand_color,'')
		FROM tournament_sponsors sp
		JOIN events e ON e.id = sp.event_id AND e.slug = ? AND e.type = 'tournament'
		WHERE sp.active = 1
		ORDER BY CASE sp.tier WHEN 'main' THEN 0 ELSE 1 END, sp.position, sp.id`
	rows, err := s.db.QueryContext(ctx, q, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Sponsor, 0, 8)
	for rows.Next() {
		var sp Sponsor
		if err := rows.Scan(&sp.ID, &sp.Name, &sp.Logo, &sp.URL, &sp.Tier, &sp.BrandColor); err != nil {
			return nil, err
		}
		out = append(out, sp)
	}
	return out, rows.Err()
}
