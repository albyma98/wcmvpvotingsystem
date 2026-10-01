package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

var errClubNotFound = errors.New("club not found")

type ClubSite struct {
	ID             int64  `json:"id"`
	OrganizationID int64  `json:"organizationId"`
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Tagline        string `json:"tagline"`
	City           string `json:"city"`
	LogoURL        string `json:"logoUrl"`
	HeroImageURL   string `json:"heroImageUrl"`
	PrimaryColor   string `json:"primaryColor"`
	SecondaryColor string `json:"secondaryColor"`
	SeasonLabel    string `json:"seasonLabel"`
	Active         bool   `json:"active"`
	CreatedAt      string `json:"createdAt"`
}

type ClubTeam struct {
	ID           int64  `json:"id"`
	SiteID       int64  `json:"siteId,omitempty"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Category     string `json:"category"`
	Gender       string `json:"gender"`
	Championship string `json:"championship"`
	Description  string `json:"description"`
	LogoURL      string `json:"logoUrl"`
	HeroImageURL string `json:"heroImageUrl"`
	Position     int    `json:"position"`
	Active       bool   `json:"active"`
}

type ClubPlayer struct {
	ID           int64  `json:"id"`
	TeamID       int64  `json:"teamId"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	JerseyNumber int    `json:"jerseyNumber"`
	Role         string `json:"role"`
	ImageURL     string `json:"imageUrl"`
	Position     int    `json:"position"`
}

type ClubPost struct {
	ID          int64  `json:"id"`
	TeamID      int64  `json:"teamId,omitempty"`
	TeamName    string `json:"teamName,omitempty"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Excerpt     string `json:"excerpt"`
	Body        string `json:"body"`
	ImageURL    string `json:"imageUrl"`
	Pinned      bool   `json:"pinned"`
	PublishedAt string `json:"publishedAt"`
	Active      bool   `json:"active"`
}

type ClubGalleryItem struct {
	ID        int64  `json:"id"`
	TeamID    int64  `json:"teamId,omitempty"`
	TeamName  string `json:"teamName,omitempty"`
	Title     string `json:"title"`
	ImageURL  string `json:"imageUrl"`
	Caption   string `json:"caption"`
	EventDate string `json:"eventDate"`
	Position  int    `json:"position"`
	Active    bool   `json:"active"`
}

type ClubMatch struct {
	ID           int64  `json:"id"`
	TeamID       int64  `json:"teamId"`
	Opponent     string `json:"opponent"`
	Competition  string `json:"competition"`
	RoundLabel   string `json:"roundLabel"`
	ScheduledAt  string `json:"scheduledAt"`
	Location     string `json:"location"`
	Home         bool   `json:"home"`
	Status       string `json:"status"`
	ScoreFor     int    `json:"scoreFor"`
	ScoreAgainst int    `json:"scoreAgainst"`
	SetsJSON     string `json:"setsJson"`
}

type ClubAdmin struct {
	ID           int64
	SiteID       int64
	Username     string
	PasswordHash string
}

func (s *Store) EnsureClubTables() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS club_sites (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL UNIQUE,
			slug TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			tagline TEXT NOT NULL DEFAULT '',
			city TEXT NOT NULL DEFAULT '',
			logo_url TEXT NOT NULL DEFAULT '',
			hero_image_url TEXT NOT NULL DEFAULT '',
			primary_color TEXT NOT NULL DEFAULT '#172554',
			secondary_color TEXT NOT NULL DEFAULT '#f59e0b',
			season_label TEXT NOT NULL DEFAULT '',
			active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS club_admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS club_admin_sessions (
			token TEXT PRIMARY KEY,
			admin_id INTEGER NOT NULL,
			site_id INTEGER NOT NULL,
			expires_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS club_teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT '',
			gender TEXT NOT NULL DEFAULT 'mixed',
			championship TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			logo_url TEXT NOT NULL DEFAULT '',
			hero_image_url TEXT NOT NULL DEFAULT '',
			position INTEGER NOT NULL DEFAULT 0,
			active INTEGER NOT NULL DEFAULT 1,
			UNIQUE(site_id, slug)
		);`,
		`CREATE TABLE IF NOT EXISTS club_players (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL,
			team_id INTEGER NOT NULL,
			first_name TEXT NOT NULL DEFAULT '',
			last_name TEXT NOT NULL DEFAULT '',
			jersey_number INTEGER NOT NULL DEFAULT 0,
			role TEXT NOT NULL DEFAULT '',
			image_url TEXT NOT NULL DEFAULT '',
			position INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS club_posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL,
			team_id INTEGER,
			kind TEXT NOT NULL DEFAULT 'announcement',
			title TEXT NOT NULL,
			excerpt TEXT NOT NULL DEFAULT '',
			body TEXT NOT NULL DEFAULT '',
			image_url TEXT NOT NULL DEFAULT '',
			pinned INTEGER NOT NULL DEFAULT 0,
			published_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			active INTEGER NOT NULL DEFAULT 1
		);`,
		`CREATE TABLE IF NOT EXISTS club_gallery (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL,
			team_id INTEGER,
			title TEXT NOT NULL DEFAULT '',
			image_url TEXT NOT NULL,
			caption TEXT NOT NULL DEFAULT '',
			event_date TEXT NOT NULL DEFAULT '',
			position INTEGER NOT NULL DEFAULT 0,
			active INTEGER NOT NULL DEFAULT 1
		);`,
		`CREATE TABLE IF NOT EXISTS club_matches (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL,
			team_id INTEGER NOT NULL,
			opponent TEXT NOT NULL,
			competition TEXT NOT NULL DEFAULT '',
			round_label TEXT NOT NULL DEFAULT '',
			scheduled_at TEXT NOT NULL,
			location TEXT NOT NULL DEFAULT '',
			home INTEGER NOT NULL DEFAULT 1,
			status TEXT NOT NULL DEFAULT 'scheduled',
			score_for INTEGER NOT NULL DEFAULT 0,
			score_against INTEGER NOT NULL DEFAULT 0,
			sets_json TEXT NOT NULL DEFAULT '[]'
		);`,
		`CREATE INDEX IF NOT EXISTS idx_club_teams_site ON club_teams(site_id, position, id);`,
		`CREATE INDEX IF NOT EXISTS idx_club_players_team ON club_players(site_id, team_id, position, id);`,
		`CREATE INDEX IF NOT EXISTS idx_club_posts_site ON club_posts(site_id, pinned DESC, published_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_club_gallery_site ON club_gallery(site_id, position, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_club_matches_team ON club_matches(site_id, team_id, scheduled_at);`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("club tables: %w", err)
		}
	}
	return nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (s *Store) ClubSiteBySlug(ctx context.Context, slug string) (ClubSite, error) {
	return scanClubSite(s.db.QueryRowContext(ctx, `SELECT id, organization_id, slug, name, tagline, city, logo_url, hero_image_url, primary_color, secondary_color, season_label, active, created_at FROM club_sites WHERE slug=?`, strings.TrimSpace(slug)))
}

func (s *Store) ClubSiteByID(ctx context.Context, id int64) (ClubSite, error) {
	return scanClubSite(s.db.QueryRowContext(ctx, `SELECT id, organization_id, slug, name, tagline, city, logo_url, hero_image_url, primary_color, secondary_color, season_label, active, created_at FROM club_sites WHERE id=?`, id))
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanClubSite(row rowScanner) (ClubSite, error) {
	var v ClubSite
	var active int
	err := row.Scan(&v.ID, &v.OrganizationID, &v.Slug, &v.Name, &v.Tagline, &v.City, &v.LogoURL, &v.HeroImageURL, &v.PrimaryColor, &v.SecondaryColor, &v.SeasonLabel, &active, &v.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return v, errClubNotFound
	}
	v.Active = active != 0
	return v, err
}

func (s *Store) ListClubSites(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.id,c.organization_id,c.slug,c.name,c.city,c.active,c.created_at,
		COALESCE(a.username,''),(SELECT COUNT(*) FROM club_teams t WHERE t.site_id=c.id),(SELECT COUNT(*) FROM club_posts p WHERE p.site_id=c.id)
		FROM club_sites c LEFT JOIN club_admins a ON a.site_id=c.id ORDER BY c.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, orgID int64
		var slug, name, city, created, username string
		var active, teams, posts int
		if err := rows.Scan(&id, &orgID, &slug, &name, &city, &active, &created, &username, &teams, &posts); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{"id": id, "organizationId": orgID, "slug": slug, "name": name, "city": city, "active": active != 0, "createdAt": created, "adminUsername": username, "teamsCount": teams, "postsCount": posts})
	}
	return out, rows.Err()
}

func (s *Store) CreateClubSite(ctx context.Context, site ClubSite, categories []string, username, passwordHash string) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO club_sites (organization_id,slug,name,tagline,city,logo_url,hero_image_url,primary_color,secondary_color,season_label,active) VALUES (?,?,?,?,?,?,?,?,?,?,1)`, site.OrganizationID, site.Slug, site.Name, site.Tagline, site.City, site.LogoURL, site.HeroImageURL, site.PrimaryColor, site.SecondaryColor, site.SeasonLabel)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	if _, err = tx.ExecContext(ctx, `INSERT INTO club_admins (site_id,username,password_hash) VALUES (?,?,?)`, id, username, passwordHash); err != nil {
		return 0, err
	}
	for i, category := range categories {
		name := strings.TrimSpace(category)
		if name == "" {
			continue
		}
		slug := slugifyTournament(name)
		gender := "mixed"
		lower := strings.ToLower(name)
		if strings.Contains(lower, "femmin") {
			gender = "female"
		} else if strings.Contains(lower, "maschil") {
			gender = "male"
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO club_teams (site_id,name,slug,category,gender,position) VALUES (?,?,?,?,?,?)`, id, name, slug, name, gender, i); err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) UpdateClubSite(ctx context.Context, site ClubSite) error {
	res, err := s.db.ExecContext(ctx, `UPDATE club_sites SET name=?,tagline=?,city=?,logo_url=?,hero_image_url=?,primary_color=?,secondary_color=?,season_label=?,active=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, site.Name, site.Tagline, site.City, site.LogoURL, site.HeroImageURL, site.PrimaryColor, site.SecondaryColor, site.SeasonLabel, boolInt(site.Active), site.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errClubNotFound
	}
	return nil
}

func (s *Store) DeleteClubSite(ctx context.Context, siteID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, table := range []string{"club_admin_sessions", "club_admins", "club_players", "club_posts", "club_gallery", "club_matches", "club_teams"} {
		if _, err = tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE site_id=?", siteID); err != nil {
			return err
		}
	}
	res, err := tx.ExecContext(ctx, "DELETE FROM club_sites WHERE id=?", siteID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errClubNotFound
	}
	return tx.Commit()
}

func (s *Store) ClubAdminByUsername(ctx context.Context, username string) (ClubAdmin, error) {
	var a ClubAdmin
	err := s.db.QueryRowContext(ctx, `SELECT id,site_id,username,password_hash FROM club_admins WHERE username=?`, strings.TrimSpace(username)).Scan(&a.ID, &a.SiteID, &a.Username, &a.PasswordHash)
	return a, err
}
func (s *Store) SetClubAdminPassword(ctx context.Context, siteID int64, hash string) (string, error) {
	var u string
	err := s.db.QueryRowContext(ctx, `SELECT username FROM club_admins WHERE site_id=?`, siteID).Scan(&u)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errClubNotFound
	}
	if err != nil {
		return "", err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE club_admins SET password_hash=? WHERE site_id=?`, hash, siteID)
	if err == nil {
		s.db.ExecContext(ctx, `DELETE FROM club_admin_sessions WHERE site_id=?`, siteID)
	}
	return u, err
}
func (s *Store) CreateClubSession(ctx context.Context, adminID, siteID int64, ttl time.Duration) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	expires := time.Now().UTC().Add(ttl).Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `INSERT INTO club_admin_sessions(token,admin_id,site_id,expires_at) VALUES(?,?,?,?)`, token, adminID, siteID, expires)
	return token, err
}
func (s *Store) ClubSession(ctx context.Context, token string) (int64, bool) {
	var siteID int64
	var exp string
	err := s.db.QueryRowContext(ctx, `SELECT site_id,expires_at FROM club_admin_sessions WHERE token=?`, token).Scan(&siteID, &exp)
	if err != nil {
		return 0, false
	}
	t, e := time.Parse(time.RFC3339Nano, exp)
	if e != nil || time.Now().After(t) {
		s.db.ExecContext(ctx, `DELETE FROM club_admin_sessions WHERE token=?`, token)
		return 0, false
	}
	return siteID, true
}
func (s *Store) DeleteClubSession(ctx context.Context, token string) {
	s.db.ExecContext(ctx, `DELETE FROM club_admin_sessions WHERE token=?`, token)
}

func (s *Store) ListClubTeams(ctx context.Context, siteID int64, activeOnly bool) ([]ClubTeam, error) {
	q := `SELECT id,site_id,name,slug,category,gender,championship,description,logo_url,hero_image_url,position,active FROM club_teams WHERE site_id=?`
	if activeOnly {
		q += ` AND active=1`
	}
	q += ` ORDER BY position,id`
	rows, err := s.db.QueryContext(ctx, q, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ClubTeam{}
	for rows.Next() {
		var v ClubTeam
		var active int
		if err := rows.Scan(&v.ID, &v.SiteID, &v.Name, &v.Slug, &v.Category, &v.Gender, &v.Championship, &v.Description, &v.LogoURL, &v.HeroImageURL, &v.Position, &active); err != nil {
			return nil, err
		}
		v.Active = active != 0
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) ClubTeamBySlug(ctx context.Context, siteID int64, slug string) (ClubTeam, error) {
	var v ClubTeam
	var active int
	err := s.db.QueryRowContext(ctx, `SELECT id,site_id,name,slug,category,gender,championship,description,logo_url,hero_image_url,position,active FROM club_teams WHERE site_id=? AND slug=?`, siteID, slug).Scan(&v.ID, &v.SiteID, &v.Name, &v.Slug, &v.Category, &v.Gender, &v.Championship, &v.Description, &v.LogoURL, &v.HeroImageURL, &v.Position, &active)
	v.Active = active != 0
	return v, err
}
func (s *Store) SaveClubTeam(ctx context.Context, siteID int64, v ClubTeam) (int64, error) {
	v.Name = strings.TrimSpace(v.Name)
	if v.Slug == "" {
		v.Slug = slugifyTournament(v.Name)
	}
	if v.ID > 0 {
		_, err := s.db.ExecContext(ctx, `UPDATE club_teams SET name=?,slug=?,category=?,gender=?,championship=?,description=?,logo_url=?,hero_image_url=?,position=?,active=? WHERE id=? AND site_id=?`, v.Name, v.Slug, v.Category, v.Gender, v.Championship, v.Description, v.LogoURL, v.HeroImageURL, v.Position, boolInt(v.Active), v.ID, siteID)
		return v.ID, err
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO club_teams(site_id,name,slug,category,gender,championship,description,logo_url,hero_image_url,position,active) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, siteID, v.Name, v.Slug, v.Category, v.Gender, v.Championship, v.Description, v.LogoURL, v.HeroImageURL, v.Position, boolInt(v.Active))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
func (s *Store) DeleteClubTeam(ctx context.Context, siteID, id int64) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	for _, t := range []string{"club_players", "club_posts", "club_gallery", "club_matches"} {
		if _, e = tx.ExecContext(ctx, "DELETE FROM "+t+" WHERE site_id=? AND team_id=?", siteID, id); e != nil {
			return e
		}
	}
	_, e = tx.ExecContext(ctx, `DELETE FROM club_teams WHERE site_id=? AND id=?`, siteID, id)
	if e != nil {
		return e
	}
	return tx.Commit()
}

func (s *Store) ListClubPlayers(ctx context.Context, siteID, teamID int64) ([]ClubPlayer, error) {
	rows, e := s.db.QueryContext(ctx, `SELECT id,team_id,first_name,last_name,jersey_number,role,image_url,position FROM club_players WHERE site_id=? AND team_id=? ORDER BY position,jersey_number,id`, siteID, teamID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []ClubPlayer{}
	for rows.Next() {
		var v ClubPlayer
		if e = rows.Scan(&v.ID, &v.TeamID, &v.FirstName, &v.LastName, &v.JerseyNumber, &v.Role, &v.ImageURL, &v.Position); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) ReplaceClubPlayers(ctx context.Context, siteID, teamID int64, items []ClubPlayer) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, `DELETE FROM club_players WHERE site_id=? AND team_id=?`, siteID, teamID); e != nil {
		return e
	}
	for i, v := range items {
		if strings.TrimSpace(v.FirstName) == "" && strings.TrimSpace(v.LastName) == "" {
			continue
		}
		if _, e = tx.ExecContext(ctx, `INSERT INTO club_players(site_id,team_id,first_name,last_name,jersey_number,role,image_url,position) VALUES(?,?,?,?,?,?,?,?)`, siteID, teamID, v.FirstName, v.LastName, v.JerseyNumber, v.Role, v.ImageURL, i); e != nil {
			return e
		}
	}
	return tx.Commit()
}

func (s *Store) ListClubPosts(ctx context.Context, siteID, teamID int64, activeOnly bool) ([]ClubPost, error) {
	q := `SELECT p.id,COALESCE(p.team_id,0),COALESCE(t.name,''),p.kind,p.title,p.excerpt,p.body,p.image_url,p.pinned,p.published_at,p.active FROM club_posts p LEFT JOIN club_teams t ON t.id=p.team_id WHERE p.site_id=?`
	args := []interface{}{siteID}
	if teamID > 0 {
		q += ` AND (p.team_id IS NULL OR p.team_id=?)`
		args = append(args, teamID)
	}
	if activeOnly {
		q += ` AND p.active=1`
	}
	q += ` ORDER BY p.pinned DESC,p.published_at DESC,p.id DESC`
	rows, e := s.db.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []ClubPost{}
	for rows.Next() {
		var v ClubPost
		var pin, active int
		if e = rows.Scan(&v.ID, &v.TeamID, &v.TeamName, &v.Kind, &v.Title, &v.Excerpt, &v.Body, &v.ImageURL, &pin, &v.PublishedAt, &active); e != nil {
			return nil, e
		}
		v.Pinned = pin != 0
		v.Active = active != 0
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) SaveClubPost(ctx context.Context, siteID int64, v ClubPost) (int64, error) {
	var team interface{}
	if v.TeamID > 0 {
		team = v.TeamID
	}
	if v.PublishedAt == "" {
		v.PublishedAt = time.Now().Format("2006-01-02")
	}
	if v.ID > 0 {
		_, e := s.db.ExecContext(ctx, `UPDATE club_posts SET team_id=?,kind=?,title=?,excerpt=?,body=?,image_url=?,pinned=?,published_at=?,active=? WHERE id=? AND site_id=?`, team, v.Kind, v.Title, v.Excerpt, v.Body, v.ImageURL, boolInt(v.Pinned), v.PublishedAt, boolInt(v.Active), v.ID, siteID)
		return v.ID, e
	}
	res, e := s.db.ExecContext(ctx, `INSERT INTO club_posts(site_id,team_id,kind,title,excerpt,body,image_url,pinned,published_at,active) VALUES(?,?,?,?,?,?,?,?,?,?)`, siteID, team, v.Kind, v.Title, v.Excerpt, v.Body, v.ImageURL, boolInt(v.Pinned), v.PublishedAt, boolInt(v.Active))
	if e != nil {
		return 0, e
	}
	return res.LastInsertId()
}

func (s *Store) ListClubGallery(ctx context.Context, siteID, teamID int64, activeOnly bool) ([]ClubGalleryItem, error) {
	q := `SELECT g.id,COALESCE(g.team_id,0),COALESCE(t.name,''),g.title,g.image_url,g.caption,g.event_date,g.position,g.active FROM club_gallery g LEFT JOIN club_teams t ON t.id=g.team_id WHERE g.site_id=?`
	args := []interface{}{siteID}
	if teamID > 0 {
		q += ` AND (g.team_id IS NULL OR g.team_id=?)`
		args = append(args, teamID)
	}
	if activeOnly {
		q += ` AND g.active=1`
	}
	q += ` ORDER BY g.position,g.id DESC`
	rows, e := s.db.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []ClubGalleryItem{}
	for rows.Next() {
		var v ClubGalleryItem
		var active int
		if e = rows.Scan(&v.ID, &v.TeamID, &v.TeamName, &v.Title, &v.ImageURL, &v.Caption, &v.EventDate, &v.Position, &active); e != nil {
			return nil, e
		}
		v.Active = active != 0
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) SaveClubGallery(ctx context.Context, siteID int64, v ClubGalleryItem) (int64, error) {
	var team interface{}
	if v.TeamID > 0 {
		team = v.TeamID
	}
	if v.ID > 0 {
		_, e := s.db.ExecContext(ctx, `UPDATE club_gallery SET team_id=?,title=?,image_url=?,caption=?,event_date=?,position=?,active=? WHERE id=? AND site_id=?`, team, v.Title, v.ImageURL, v.Caption, v.EventDate, v.Position, boolInt(v.Active), v.ID, siteID)
		return v.ID, e
	}
	res, e := s.db.ExecContext(ctx, `INSERT INTO club_gallery(site_id,team_id,title,image_url,caption,event_date,position,active) VALUES(?,?,?,?,?,?,?,?)`, siteID, team, v.Title, v.ImageURL, v.Caption, v.EventDate, v.Position, boolInt(v.Active))
	if e != nil {
		return 0, e
	}
	return res.LastInsertId()
}

func (s *Store) ListClubMatches(ctx context.Context, siteID, teamID int64) ([]ClubMatch, error) {
	q := `SELECT id,team_id,opponent,competition,round_label,scheduled_at,location,home,status,score_for,score_against,sets_json FROM club_matches WHERE site_id=?`
	args := []interface{}{siteID}
	if teamID > 0 {
		q += ` AND team_id=?`
		args = append(args, teamID)
	}
	q += ` ORDER BY scheduled_at DESC,id DESC`
	rows, e := s.db.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []ClubMatch{}
	for rows.Next() {
		var v ClubMatch
		var home int
		if e = rows.Scan(&v.ID, &v.TeamID, &v.Opponent, &v.Competition, &v.RoundLabel, &v.ScheduledAt, &v.Location, &home, &v.Status, &v.ScoreFor, &v.ScoreAgainst, &v.SetsJSON); e != nil {
			return nil, e
		}
		v.Home = home != 0
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) SaveClubMatch(ctx context.Context, siteID int64, v ClubMatch) (int64, error) {
	if v.Status == "" {
		v.Status = "scheduled"
	}
	if v.SetsJSON == "" {
		v.SetsJSON = "[]"
	}
	if v.ID > 0 {
		_, e := s.db.ExecContext(ctx, `UPDATE club_matches SET team_id=?,opponent=?,competition=?,round_label=?,scheduled_at=?,location=?,home=?,status=?,score_for=?,score_against=?,sets_json=? WHERE id=? AND site_id=?`, v.TeamID, v.Opponent, v.Competition, v.RoundLabel, v.ScheduledAt, v.Location, boolInt(v.Home), v.Status, v.ScoreFor, v.ScoreAgainst, v.SetsJSON, v.ID, siteID)
		return v.ID, e
	}
	res, e := s.db.ExecContext(ctx, `INSERT INTO club_matches(site_id,team_id,opponent,competition,round_label,scheduled_at,location,home,status,score_for,score_against,sets_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, siteID, v.TeamID, v.Opponent, v.Competition, v.RoundLabel, v.ScheduledAt, v.Location, boolInt(v.Home), v.Status, v.ScoreFor, v.ScoreAgainst, v.SetsJSON)
	if e != nil {
		return 0, e
	}
	return res.LastInsertId()
}

func (s *Store) DeleteClubRecord(ctx context.Context, siteID int64, table string, id int64) error {
	allowed := map[string]bool{"club_posts": true, "club_gallery": true, "club_matches": true}
	if !allowed[table] {
		return errors.New("invalid table")
	}
	_, e := s.db.ExecContext(ctx, "DELETE FROM "+table+" WHERE site_id=? AND id=?", siteID, id)
	return e
}
