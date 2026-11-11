package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSponsorPlacementsScopeAndLifecycle(t *testing.T) {
	conn, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "walls.db"))+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(`
		CREATE TABLE organizations(id INTEGER PRIMARY KEY);
		CREATE TABLE sponsors(id INTEGER PRIMARY KEY, organization_id INTEGER, is_active INTEGER);
		INSERT INTO organizations VALUES (1), (2);
		INSERT INTO sponsors VALUES (12, 1, 1), (13, 1, 1), (21, 1, 0), (34, 2, 1);
	`); err != nil {
		t.Fatal(err)
	}
	if err := ensureSponsorPlacements(conn); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	db := &appdbimpl{c: conn}
	check := func(org int, group SponsorPlacementGroup, activeOnly bool, want map[string]int) {
		t.Helper()
		got, err := db.GetSponsorPlacements(ctx, org, group, activeOnly)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("org %d %s active %t: got %v, want %v, err %v", org, group, activeOnly, got, want, err)
		}
	}
	save := func(org int, group SponsorPlacementGroup, assignments map[string]int) {
		t.Helper()
		if err := db.SaveSponsorPlacements(ctx, org, group, assignments); err != nil {
			t.Fatal(err)
		}
	}

	assignments := map[string]int{"basketball_left": 12, "playmaker_right": 21, "crossover_right": 12}
	mvpAssignments := map[string]int{"post_vote_left": 12, "post_vote_right": 13}
	save(1, GameWallPlacements, assignments)
	save(2, GameWallPlacements, map[string]int{"basketball_left": 34})
	save(1, PostVotePlacements, mvpAssignments)
	save(2, PostVotePlacements, map[string]int{"post_vote_right": 34})
	if err := ensureSponsorPlacements(conn); err != nil {
		t.Fatal(err)
	}
	check(1, GameWallPlacements, false, assignments)
	check(1, GameWallPlacements, true, map[string]int{"basketball_left": 12, "crossover_right": 12})
	check(2, GameWallPlacements, true, map[string]int{"basketball_left": 34})
	check(1, PostVotePlacements, true, mvpAssignments)
	check(2, PostVotePlacements, true, map[string]int{"post_vote_right": 34})
	for _, invalid := range []map[string]int{
		{"basketball_right": 34},
		{"basketball_right": 99},
		{"volley_left": 12},
		{"basketball_left": -1},
	} {
		if err := db.SaveSponsorPlacements(ctx, 1, GameWallPlacements, invalid); !errors.Is(err, ErrInvalidSponsorPlacement) {
			t.Fatalf("invalid assignment %v: %v", invalid, err)
		}
		check(1, GameWallPlacements, false, assignments)
	}
	for _, invalid := range []map[string]int{
		{"post_vote_left": 13, "post_vote_right": 34},
		{"basketball_left": 12},
	} {
		if err := db.SaveSponsorPlacements(ctx, 1, PostVotePlacements, invalid); !errors.Is(err, ErrInvalidSponsorPlacement) {
			t.Fatalf("invalid post-vote assignment %v: %v", invalid, err)
		}
		check(1, PostVotePlacements, false, mvpAssignments)
	}
	if _, err := conn.Exec(`UPDATE sponsors SET is_active=1 WHERE id=21; UPDATE sponsors SET is_active=0 WHERE id=13; DELETE FROM sponsors WHERE id=12`); err != nil {
		t.Fatal(err)
	}
	check(1, GameWallPlacements, true, map[string]int{"playmaker_right": 21})
	check(1, PostVotePlacements, false, map[string]int{"post_vote_right": 13})
	check(1, PostVotePlacements, true, map[string]int{})
	save(1, PostVotePlacements, map[string]int{"post_vote_right": 0})
	check(1, PostVotePlacements, false, map[string]int{})
	check(1, GameWallPlacements, true, map[string]int{"playmaker_right": 21})
	save(1, GameWallPlacements, map[string]int{"playmaker_right": 0})
	check(1, GameWallPlacements, true, map[string]int{})
	check(2, GameWallPlacements, true, map[string]int{"basketball_left": 34})
	check(2, PostVotePlacements, true, map[string]int{"post_vote_right": 34})
}
