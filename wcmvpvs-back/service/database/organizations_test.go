package database

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestListOrganizationsWithNullOptionalFields(t *testing.T) {
	conn, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "organizations.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	appdb, err := New(conn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`INSERT INTO organizations (name, slug, city, logo_url) VALUES
		('Club senza dettagli', 'senza-dettagli', NULL, NULL),
		('Club completo', 'completo', 'Roma', 'https://example.com/logo.png')`); err != nil {
		t.Fatal(err)
	}

	orgs, err := appdb.ListOrganizations()
	if err != nil {
		t.Fatalf("list organizations: %v", err)
	}
	if len(orgs) != 2 {
		t.Fatalf("got %d organizations, want 2", len(orgs))
	}
	if orgs[0].City != "Roma" || orgs[0].LogoURL != "https://example.com/logo.png" {
		t.Fatalf("populated optional fields lost: %+v", orgs[0])
	}
	if orgs[1].City != "" || orgs[1].LogoURL != "" {
		t.Fatalf("NULL optional fields not returned as empty strings: %+v", orgs[1])
	}

	byID, err := appdb.GetOrganization(orgs[1].ID)
	if err != nil || byID.City != "" || byID.LogoURL != "" {
		t.Fatalf("get organization by ID: %+v, %v", byID, err)
	}
	bySlug, err := appdb.GetOrganizationBySlug("senza-dettagli")
	if err != nil || bySlug.ID != orgs[1].ID {
		t.Fatalf("get organization by slug: %+v, %v", bySlug, err)
	}
}
