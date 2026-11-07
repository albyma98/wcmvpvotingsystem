package database

import (
	"bytes"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrInvalidArenaData = errors.New("invalid arena data")
	ErrArenaSlugExists  = errors.New("arena slug already exists")
	ErrInvalidArenaID   = errors.New("invalid arena ID")
	arenaSlugPattern    = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

//go:embed arena_scenes/baltur-arena.json
var balturArenaScene string

type ArenaSummary struct {
	ID        int    `json:"id"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	City      string `json:"city"`
	TeamName  string `json:"team_name"`
	Capacity  *int   `json:"capacity"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Arena struct {
	ArenaSummary
	SceneConfig json.RawMessage `json:"scene_config"`
}

const arenaSummaryColumns = `a.id, a.slug, a.name, a.city, a.team_name, a.capacity, a.is_active, a.created_at, a.updated_at`
const arenaForOrganization = `FROM organizations o JOIN arenas a ON a.id = o.arena_id WHERE o.id = ?`

func ensureArenaCatalog(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS arenas (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		slug TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		city TEXT NOT NULL,
		team_name TEXT NOT NULL DEFAULT '',
		capacity INTEGER CHECK (capacity > 0),
		is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
		scene_config_json TEXT NOT NULL,
		seed_key TEXT UNIQUE,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE organizations ADD COLUMN arena_id INTEGER REFERENCES arenas(id)`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return err
	}

	// Keep seed identity stable when a master edits the slug or other fields.
	if _, err := tx.Exec(`WITH seeds (slug, name, city, team_name, capacity, scene_config_json) AS (VALUES
		('joy-volley-arena', 'Joy Volley Arena', 'Gioia del Colle', 'Joy Volley', NULL, '{"renderer":"joy-volley"}'),
		('baltur-arena', 'Baltur Arena', 'Cento', 'Benedetto XIV Cento', 2000, ?)
	)
	INSERT INTO arenas (seed_key, slug, name, city, team_name, capacity, scene_config_json)
	SELECT s.slug, s.slug, s.name, s.city, s.team_name, s.capacity, s.scene_config_json FROM seeds s
	WHERE NOT EXISTS (SELECT 1 FROM arenas a WHERE a.seed_key = s.slug OR a.slug = s.slug)`, balturArenaScene); err != nil {
		return err
	}
	return tx.Commit()
}

func (a *ArenaSummary) scanFields() []interface{} {
	return []interface{}{&a.ID, &a.Slug, &a.Name, &a.City, &a.TeamName, &a.Capacity, &a.IsActive, &a.CreatedAt, &a.UpdatedAt}
}

func scanArena(row rowScanner) (Arena, error) {
	var arena Arena
	var config string
	if err := row.Scan(append(arena.scanFields(), &config)...); err != nil {
		return Arena{}, err
	}
	arena.SceneConfig = json.RawMessage(config)
	return arena, nil
}

func (db *appdbimpl) ListArenas() ([]ArenaSummary, error) {
	rows, err := db.c.Query(`SELECT ` + arenaSummaryColumns + ` FROM arenas a ORDER BY a.name COLLATE NOCASE, a.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	arenas := []ArenaSummary{}
	for rows.Next() {
		var arena ArenaSummary
		if err := rows.Scan(arena.scanFields()...); err != nil {
			return nil, err
		}
		arenas = append(arenas, arena)
	}
	return arenas, rows.Err()
}

func (db *appdbimpl) GetArena(id int) (Arena, error) {
	return scanArena(db.c.QueryRow(`SELECT `+arenaSummaryColumns+`, a.scene_config_json FROM arenas a WHERE a.id = ?`, id))
}

func (db *appdbimpl) GetArenaByOrganization(organizationID int) (Arena, error) {
	return scanArena(db.c.QueryRow(`SELECT `+arenaSummaryColumns+`, a.scene_config_json `+arenaForOrganization, organizationID))
}

func normalizeArena(arena *Arena) error {
	arena.Name = strings.TrimSpace(arena.Name)
	arena.Slug = strings.TrimSpace(arena.Slug)
	arena.City = strings.TrimSpace(arena.City)
	arena.TeamName = strings.TrimSpace(arena.TeamName)
	arena.SceneConfig = bytes.TrimSpace(arena.SceneConfig)
	if arena.Name == "" || arena.City == "" || !arenaSlugPattern.MatchString(arena.Slug) {
		return ErrInvalidArenaData
	}
	if arena.Capacity != nil && *arena.Capacity <= 0 {
		return ErrInvalidArenaData
	}
	if len(arena.SceneConfig) == 0 || len(arena.SceneConfig) > 256*1024 || arena.SceneConfig[0] != '{' || !json.Valid(arena.SceneConfig) {
		return ErrInvalidArenaData
	}
	return nil
}

func arenaWriteError(err error) error {
	if strings.Contains(err.Error(), "UNIQUE constraint failed: arenas.slug") {
		return ErrArenaSlugExists
	}
	return err
}

func (db *appdbimpl) CreateArena(arena Arena) (Arena, error) {
	if err := normalizeArena(&arena); err != nil {
		return Arena{}, err
	}
	res, err := db.c.Exec(`INSERT INTO arenas (slug, name, city, team_name, capacity, is_active, scene_config_json) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		arena.Slug, arena.Name, arena.City, arena.TeamName, arena.Capacity, arena.IsActive, string(arena.SceneConfig))
	if err != nil {
		return Arena{}, arenaWriteError(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Arena{}, err
	}
	return db.GetArena(int(id))
}

func (db *appdbimpl) UpdateArena(arena Arena) (Arena, error) {
	if err := normalizeArena(&arena); err != nil {
		return Arena{}, err
	}
	res, err := db.c.Exec(`UPDATE arenas SET slug=?, name=?, city=?, team_name=?, capacity=?, is_active=?, scene_config_json=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		arena.Slug, arena.Name, arena.City, arena.TeamName, arena.Capacity, arena.IsActive, string(arena.SceneConfig), arena.ID)
	if err != nil {
		return Arena{}, arenaWriteError(err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return Arena{}, err
	} else if n == 0 {
		return Arena{}, sql.ErrNoRows
	}
	return db.GetArena(arena.ID)
}

func validateArenaAssignment(tx *sql.Tx, arenaID *int) error {
	if arenaID == nil {
		return nil
	}
	var exists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM arenas WHERE id = ?)`, *arenaID).Scan(&exists); err != nil {
		return fmt.Errorf("validating arena assignment: %w", err)
	}
	if !exists {
		return ErrInvalidArenaID
	}
	return nil
}

func (db *appdbimpl) SetOrganizationArena(organizationID int, arenaID *int) (Organization, error) {
	tx, err := db.c.Begin()
	if err != nil {
		return Organization{}, err
	}
	defer tx.Rollback()
	if err := validateArenaAssignment(tx, arenaID); err != nil {
		return Organization{}, err
	}
	res, err := tx.Exec(`UPDATE organizations SET arena_id = ? WHERE id = ?`, arenaID, organizationID)
	if err != nil {
		return Organization{}, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return Organization{}, err
	} else if n == 0 {
		return Organization{}, sql.ErrNoRows
	}
	if err := tx.Commit(); err != nil {
		return Organization{}, err
	}
	return db.GetOrganization(organizationID)
}
