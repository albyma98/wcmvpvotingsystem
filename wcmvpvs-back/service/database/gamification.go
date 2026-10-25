package database

import (
	"database/sql"
	"encoding/json"
	"errors"
)

var ErrInvalidMinigameSports = errors.New("invalid minigame sports")

type Minigame struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Sports      []string `json:"sports"`
}

// The same game shares its sport rules across the standard and sponsor versions.
func MinigameID(id string) string {
	switch id {
	case "tap_challenge":
		return "tap"
	case "memory_flash":
		return "memory-flash"
	case "sponsor_rush":
		return "sponsor-rush"
	default:
		return id
	}
}

func minigameCatalog() []Minigame {
	return []Minigame{
		{ID: "reaction", Name: "Reaction Test", Description: "Test dei riflessi e tempi di reazione."},
		{ID: "quiz", Name: "Quiz Lampo", Description: "Domande sul match; richiede il quiz attivo per l’evento."},
		{ID: "basketball", Name: "Basket Rush", Description: "Sfida di tiri a canestro."},
		{ID: "playmaker", Name: "Playmaker", Description: "Assist e passaggi a tempo."},
		{ID: "crossover", Name: "Crossover Rush", Description: "Finte, cambi di mano e schiacciate."},
		{ID: "tap", Name: "Tap Challenge / Tap Battle", Description: "Sfida di tocchi, anche in modalità live e sponsor."},
		{ID: "sponsor-rush", Name: "Sponsor Rush", Description: "Raccogli i loghi degli sponsor; include la versione sponsor."},
		{ID: "memory-flash", Name: "Memory Flash", Description: "Trova le coppie; include la versione sponsor."},
		{ID: "free_throw_challenge", Name: "Free Throw Challenge", Description: "Sfida sponsor di tiri liberi."},
		{ID: "stack_it", Name: "Stack It", Description: "Costruisci la torre sponsorizzata."},
		{ID: "libero-reflex", Name: "Libero Reflex", Description: "Blocca la palla nella zona verde dopo il voto."},
		{ID: "volley-rush", Name: "Volley Rush", Description: "Mira, supera il muro e schiaccia: un punto, una moneta.", Sports: []string{"Pallavolo"}},
		{ID: "setter-pro", Name: "Setter Pro", Description: "Un gesto per l’alzata perfetta e l’assist allo schiacciatore.", Sports: []string{"Pallavolo"}},
		{ID: "monster-block", Name: "Monster Block", Description: "Leggi l’attacco, chiudi a rete e firma il muro vincente.", Sports: []string{"Pallavolo"}},
	}
}

func ensureGamificationTables(conn *sql.DB) error {
	_, err := conn.Exec(`CREATE TABLE IF NOT EXISTS minigame_sports (game_id TEXT PRIMARY KEY, sports_json TEXT NOT NULL)`)
	return err
}

func (db *appdbimpl) ListMinigames() ([]Minigame, error) {
	games := minigameCatalog()
	rules := map[string][]string{}
	rows, err := db.c.Query(`SELECT game_id,sports_json FROM minigame_sports`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, raw string
		var sports []string
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &sports); err != nil {
			return nil, err
		}
		rules[id] = sports
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range games {
		if games[i].Sports == nil {
			games[i].Sports = []string{"Tutti"}
		}
		if sports, ok := rules[games[i].ID]; ok {
			games[i].Sports = sports
		}
	}
	return games, nil
}

func (db *appdbimpl) UpdateMinigameSports(id string, sports []string) (Minigame, error) {
	var game Minigame
	for _, entry := range minigameCatalog() {
		if entry.ID == id {
			game = entry
			break
		}
	}
	if game.ID == "" {
		return game, sql.ErrNoRows
	}
	selected := map[string]bool{}
	for _, sport := range sports {
		switch sport {
		case "Tutti", "Calcio", "Pallavolo", "Pallacanestro":
			selected[sport] = true
		default:
			return game, ErrInvalidMinigameSports
		}
	}
	if selected["Tutti"] && len(selected) != 1 {
		return game, ErrInvalidMinigameSports
	}
	game.Sports = []string{}
	for _, sport := range []string{"Tutti", "Calcio", "Pallavolo", "Pallacanestro"} {
		if selected[sport] {
			game.Sports = append(game.Sports, sport)
		}
	}
	raw, err := json.Marshal(game.Sports)
	if err != nil {
		return game, err
	}
	_, err = db.c.Exec(`INSERT INTO minigame_sports(game_id,sports_json) VALUES(?,?) ON CONFLICT(game_id) DO UPDATE SET sports_json=excluded.sports_json`, id, string(raw))
	return game, err
}

func (db *appdbimpl) GetOrganizationMinigames(organizationID int) ([]string, error) {
	org, err := db.GetOrganization(organizationID)
	if err != nil {
		return nil, err
	}
	games, err := db.ListMinigames()
	if err != nil {
		return nil, err
	}
	available := []string{}
	for _, game := range games {
		for _, sport := range game.Sports {
			if sport == "Tutti" || (org.Sport != "" && sport == org.Sport) {
				available = append(available, game.ID)
				break
			}
		}
	}
	return available, nil
}
