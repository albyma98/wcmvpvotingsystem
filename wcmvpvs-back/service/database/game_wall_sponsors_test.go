package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGameWallSponsorsScopeAndLifecycle(t *testing.T) {
	conn, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "walls.db"))+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(`
		CREATE TABLE organizations(id INTEGER PRIMARY KEY);
		CREATE TABLE sponsors(id INTEGER PRIMARY KEY, organization_id INTEGER, is_active INTEGER);
		INSERT INTO organizations VALUES (1), (2);
		INSERT INTO sponsors VALUES (12, 1, 1), (21, 1, 0), (34, 2, 1);
	`); err != nil {
		t.Fatal(err)
	}
	if err := ensureGameWallSponsors(conn); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	db := &appdbimpl{c: conn}
	check := func(org int, activeOnly bool, want map[string]int) {
		t.Helper()
		got, err := db.GetGameWallSponsors(ctx, org, activeOnly)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("org %d active %t: got %v, want %v, err %v", org, activeOnly, got, want, err)
		}
	}

	assignments := map[string]int{"basketball_left": 12, "playmaker_right": 21, "crossover_right": 12}
	if err := db.SaveGameWallSponsors(ctx, 1, assignments); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGameWallSponsors(ctx, 2, map[string]int{"basketball_left": 34}); err != nil {
		t.Fatal(err)
	}
	check(1, false, assignments)
	check(1, true, map[string]int{"basketball_left": 12, "crossover_right": 12})
	check(2, true, map[string]int{"basketball_left": 34})
	for _, invalid := range []map[string]int{
		{"basketball_right": 34},
		{"basketball_right": 99},
		{"volley_left": 12},
		{"basketball_left": -1},
	} {
		if err := db.SaveGameWallSponsors(ctx, 1, invalid); !errors.Is(err, ErrInvalidGameWallSponsor) {
			t.Fatalf("invalid assignment %v: %v", invalid, err)
		}
		check(1, false, assignments)
	}
	if _, err := conn.Exec(`UPDATE sponsors SET is_active=1 WHERE id=21; DELETE FROM sponsors WHERE id=12`); err != nil {
		t.Fatal(err)
	}
	check(1, true, map[string]int{"playmaker_right": 21})
	if err := db.SaveGameWallSponsors(ctx, 1, map[string]int{"playmaker_right": 0}); err != nil {
		t.Fatal(err)
	}
	check(1, true, map[string]int{})
	check(2, true, map[string]int{"basketball_left": 34})
}
