package api

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestClubStoreProvisionAndPublicData(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "club.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := NewStore(db)
	if err := store.EnsureClubTables(); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	ctx := context.Background()
	siteID, err := store.CreateClubSite(ctx, ClubSite{
		OrganizationID: 1, Slug: "aurora-volley", Name: "Aurora Volley",
		PrimaryColor: "#172554", SecondaryColor: "#f59e0b", Active: true,
	}, []string{"Prima Divisione Maschile", "Under 16"}, "club-aurora-volley", "hash")
	if err != nil {
		t.Fatalf("create site: %v", err)
	}
	if siteID <= 0 {
		t.Fatal("missing site id")
	}
	site, err := store.ClubSiteBySlug(ctx, "aurora-volley")
	if err != nil || site.Name != "Aurora Volley" {
		t.Fatalf("site: %#v err=%v", site, err)
	}
	teams, err := store.ListClubTeams(ctx, siteID, true)
	if err != nil || len(teams) != 2 {
		t.Fatalf("teams=%d err=%v", len(teams), err)
	}
	if err := store.ReplaceClubPlayers(ctx, siteID, teams[0].ID, []ClubPlayer{{FirstName: "Mario", LastName: "Rossi", JerseyNumber: 7, Role: "Schiacciatore"}}); err != nil {
		t.Fatalf("players: %v", err)
	}
	players, _ := store.ListClubPlayers(ctx, siteID, teams[0].ID)
	if len(players) != 1 || players[0].JerseyNumber != 7 {
		t.Fatalf("players: %#v", players)
	}
	if _, err := store.SaveClubPost(ctx, siteID, ClubPost{Title: "Open day", Kind: "announcement", Active: true}); err != nil {
		t.Fatalf("post: %v", err)
	}
	posts, _ := store.ListClubPosts(ctx, siteID, 0, true)
	if len(posts) != 1 || posts[0].Title != "Open day" {
		t.Fatalf("posts: %#v", posts)
	}
}
