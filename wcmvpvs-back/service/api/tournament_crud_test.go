package api

// Verifica end-to-end del CRUD torneo con le FK ATTIVE (come in produzione:
// PRAGMA foreign_keys=ON). Riproduce e blocca la regressione
// "FOREIGN KEY constraint failed" su creazione torneo e su creazione partita.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	_ "github.com/mattn/go-sqlite3"
)

func TestTournamentCRUD_ForeignKeysEnabled(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "crud.db") + "?_foreign_keys=on"
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		t.Fatalf("pragma: %v", err)
	}

	appdb, err := database.New(conn)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db := appdb.SQLConn()
	store := NewStore(db)
	if err := store.EnsureTournamentAdminTables(); err != nil {
		t.Fatalf("EnsureTournamentAdminTables: %v", err)
	}

	ctx := context.Background()

	// 1) CREATE tournament (era il caso che falliva con FK constraint failed)
	evID, err := store.CreateTournament(ctx, TournamentCreateInput{
		Name: "Test Cup", Format: "BEACH VOLLEY 4X4", DateLabel: "1-2 GIU 2026",
		Location: "Roma", Slug: "test-cup",
	}, "admin", "hash")
	if err != nil {
		t.Fatalf("CreateTournament: %v", err)
	}

	// Creare due volte deve riusare le sentinelle (una sola org/team di sistema)
	if _, err := store.CreateTournament(ctx, TournamentCreateInput{
		Name: "Test Cup 2", Format: "4x4", DateLabel: "x", Location: "Milano", Slug: "test-cup-2",
	}, "admin2", "hash2"); err != nil {
		t.Fatalf("CreateTournament (second): %v", err)
	}
	var orgCount, teamCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM organizations WHERE slug='__tournament_sys__'`).Scan(&orgCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM teams WHERE championship='__tournament_sys__'`).Scan(&teamCount)
	if orgCount != 1 || teamCount != 1 {
		t.Fatalf("sentinelle non condivise: org=%d team=%d (attese 1 e 1)", orgCount, teamCount)
	}

	// 2) CREATE squadre torneo + 3) CREATE partita (FK matches.team_* -> tournament_teams)
	var teamA, teamB int64
	if res, err := db.Exec(`INSERT INTO tournament_teams (event_id, name) VALUES (?, 'Mambo Beach')`, evID); err != nil {
		t.Fatalf("insert team A: %v", err)
	} else {
		teamA, _ = res.LastInsertId()
	}
	if res, err := db.Exec(`INSERT INTO tournament_teams (event_id, name) VALUES (?, 'Netbreakers')`, evID); err != nil {
		t.Fatalf("insert team B: %v", err)
	} else {
		teamB, _ = res.LastInsertId()
	}

	matchID, err := store.CreateTAMatch(ctx, evID, "CAMPO 1", "18:30", "", "", teamA, teamB)
	if err != nil {
		t.Fatalf("CreateTAMatch: %v", err)
	}

	// 4) UPDATE punteggio (console operatore)
	if err := store.ApplyScoreAction(ctx, evID, matchID, "start"); err != nil {
		t.Fatalf("ApplyScoreAction start: %v", err)
	}
	if err := store.ApplyScoreAction(ctx, evID, matchID, "point_a"); err != nil {
		t.Fatalf("ApplyScoreAction point_a: %v", err)
	}
	if err := store.ApplyScoreAction(ctx, evID, matchID, "close_set"); err != nil {
		t.Fatalf("ApplyScoreAction close_set: %v", err)
	}
	if err := store.ApplyScoreAction(ctx, evID, matchID, "undo_last_set"); err != nil {
		t.Fatalf("ApplyScoreAction undo_last_set: %v", err)
	}
	var status, setLabel, setsJSON string
	var scoreA, scoreB, curA, curB int
	if err := db.QueryRow(`
		SELECT status, set_label, sets_json, score_a, score_b, cur_a, cur_b
		FROM matches WHERE id = ?`, matchID).
		Scan(&status, &setLabel, &setsJSON, &scoreA, &scoreB, &curA, &curB); err != nil {
		t.Fatalf("read score after undo_last_set: %v", err)
	}
	if status != "live" || setLabel != "1° SET" || setsJSON != "[]" ||
		scoreA != 0 || scoreB != 0 || curA != 1 || curB != 0 {
		t.Fatalf("undo_last_set state: status=%q label=%q sets=%q score=%d:%d current=%d:%d",
			status, setLabel, setsJSON, scoreA, scoreB, curA, curB)
	}

	// 5) Le modalita MVP pubblico/organizzatore sono indipendenti e arrivano
	// correttamente fino alla home pubblica.
	settings, _, err := store.GetTASettings(ctx, evID)
	if err != nil {
		t.Fatalf("GetTASettings: %v", err)
	}
	settings.MvpByGender = true
	settings.OrganizerMvpByGender = false
	settings.Prizes.OrgMvp = "Premio MVP assoluto"
	if err := store.UpdateTASettings(ctx, evID, *settings); err != nil {
		t.Fatalf("UpdateTASettings MVP modes: %v", err)
	}
	savedSettings, _, err := store.GetTASettings(ctx, evID)
	if err != nil {
		t.Fatalf("GetTASettings after update: %v", err)
	}
	if !savedSettings.MvpByGender || savedSettings.OrganizerMvpByGender {
		t.Fatalf("modalita MVP non indipendenti: pubblico=%v organizzatore=%v",
			savedSettings.MvpByGender, savedSettings.OrganizerMvpByGender)
	}

	// 6) READ pubblico (join su tournament_teams + matches)
	home, err := store.GetTournamentHome(ctx, "test-cup")
	if err != nil {
		t.Fatalf("GetTournamentHome: %v", err)
	}
	if !home.Tournament.MvpByGender || home.Tournament.OrganizerMvpByGender {
		t.Fatalf("modalita MVP home errate: pubblico=%v organizzatore=%v",
			home.Tournament.MvpByGender, home.Tournament.OrganizerMvpByGender)
	}
	if _, err := store.GetTournamentLive(ctx, "test-cup"); err != nil {
		t.Fatalf("GetTournamentLive: %v", err)
	}

	// 7) SHOP: persistenza admin e inclusione nello snapshot pubblico.
	productID, err := store.CreateTAShopProduct(ctx, evID, TournamentShopProduct{
		ImageURL:   "data:image/webp;base64,dGVzdA==",
		Title:      "Panino",
		PriceCents: 750,
		Extras: []TournamentShopExtra{
			{Title: "Bacon", PriceCents: 150},
		},
	})
	if err != nil {
		t.Fatalf("CreateTAShopProduct: %v", err)
	}
	shop, err := store.ListTAShopProducts(ctx, evID)
	if err != nil || len(shop) != 1 || len(shop[0].Extras) != 1 {
		t.Fatalf("ListTAShopProducts: products=%+v err=%v", shop, err)
	}
	home, err = store.GetTournamentHome(ctx, "test-cup")
	if err != nil || len(home.ShopProducts) != 1 || home.ShopProducts[0].PriceCents != 750 {
		t.Fatalf("GetTournamentHome shop: home=%+v err=%v", home, err)
	}
	if _, err := store.CreateTournamentShopReservation(
		ctx, "test-cup", productID, []string{"Bacon"}, "Mario", "Rossi", "+39 333 1234567",
	); err != nil {
		t.Fatalf("CreateTournamentShopReservation: %v", err)
	}
	reservations, err := store.ListTAShopReservations(ctx, evID)
	if err != nil || len(reservations) != 1 || reservations[0].TotalPriceCents != 900 {
		t.Fatalf("ListTAShopReservations: reservations=%+v err=%v", reservations, err)
	}
	if err := store.DeleteTAShopProduct(ctx, evID, productID); err != nil {
		t.Fatalf("DeleteTAShopProduct: %v", err)
	}

	// 8) DELETE partita
	if err := store.DeleteTAMatch(ctx, evID, matchID); err != nil {
		t.Fatalf("DeleteTAMatch: %v", err)
	}
}
