package database

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
)

func TestGamificationSportsPersistAndFollowOrganizationSport(t *testing.T) {
	db, club, _, _, _ := fanEconomyFixture(t)
	games, err := db.ListMinigames()
	if err != nil || len(games) != 11 {
		t.Fatalf("catalog: %+v %v", games, err)
	}
	for _, game := range games {
		if !reflect.DeepEqual(game.Sports, []string{"Tutti"}) {
			t.Fatalf("default availability changed: %+v", game)
		}
	}
	if _, err := db.UpdateMinigameSports("basketball", []string{"Pallacanestro"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpdateMinigameSports("reaction", []string{"Pallavolo", "Calcio", "Calcio"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpdateMinigameSports("quiz", []string{}); err != nil {
		t.Fatal(err)
	}
	// Re-running the startup migration must preserve the master configuration.
	app, err := New(db.c)
	if err != nil {
		t.Fatal(err)
	}
	games, err = app.ListMinigames()
	if err != nil || !reflect.DeepEqual(games[0].Sports, []string{"Calcio", "Pallavolo"}) || len(games[1].Sports) != 0 {
		t.Fatalf("persisted catalog: %+v %v", games, err)
	}
	for _, sport := range []string{"", "Calcio", "Pallavolo", "Pallacanestro"} {
		club.Sport = sport
		if _, err := db.UpdateOrganization(club); err != nil {
			t.Fatal(err)
		}
		available, err := app.GetOrganizationMinigames(club.ID)
		if err != nil {
			t.Fatal(err)
		}
		has := func(id string) bool {
			for _, game := range available {
				if id == game {
					return true
				}
			}
			return false
		}
		if has("basketball") != (sport == "Pallacanestro") || has("reaction") != (sport == "Calcio" || sport == "Pallavolo") || has("quiz") || !has("tap") {
			t.Fatalf("sport %q: %+v", sport, available)
		}
	}
	if _, err := db.UpdateMinigameSports("basketball", []string{"Tutti"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetOrganizationMinigames(999999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unknown organization: %v", err)
	}
}

func TestGamificationRejectsInvalidAssignments(t *testing.T) {
	db, _, _, _, _ := fanEconomyFixture(t)
	for _, sports := range [][]string{{"Tennis"}, {"Tutti", "Calcio"}, {""}, {"calcio"}} {
		if _, err := db.UpdateMinigameSports("basketball", sports); !errors.Is(err, ErrInvalidMinigameSports) {
			t.Fatalf("accepted %+v: %v", sports, err)
		}
	}
	if _, err := db.UpdateMinigameSports("unknown", []string{"Tutti"}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unknown game: %v", err)
	}
	for alias, canonical := range map[string]string{"tap_challenge": "tap", "memory_flash": "memory-flash", "sponsor_rush": "sponsor-rush", "stack_it": "stack_it"} {
		if MinigameID(alias) != canonical {
			t.Fatalf("alias %s", alias)
		}
	}
}
