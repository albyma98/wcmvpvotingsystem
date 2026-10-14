package database

import (
	"database/sql"
	"errors"
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

func TestOrganizationColorsPersistenceAndValidation(t *testing.T) {
	conn, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "colors.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	// Simulate a database created before organization colors were introduced.
	if _, err := conn.Exec(`CREATE TABLE organizations (
		id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, slug TEXT NOT NULL DEFAULT '',
		city TEXT, logo_url TEXT, is_active INTEGER NOT NULL DEFAULT 1, roster_schema INTEGER NOT NULL DEFAULT 13,
		team_id INTEGER, sms_cost REAL NOT NULL DEFAULT 0.08, free_sms INTEGER NOT NULL DEFAULT 0,
		bar_enabled INTEGER NOT NULL DEFAULT 1, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
		INSERT INTO organizations (name, slug) VALUES ('Existing', 'existing')`); err != nil {
		t.Fatal(err)
	}
	appdb, err := New(conn)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := appdb.GetOrganizationBySlug("existing")
	if err != nil {
		t.Fatal(err)
	}
	if legacy.PrimaryColor != DefaultOrganizationPrimaryColor || legacy.SecondaryColor != DefaultOrganizationSecondaryColor || legacy.TertiaryColor != DefaultOrganizationTertiaryColor {
		t.Fatalf("legacy defaults: %+v", legacy)
	}
	// Running startup migrations again must be safe.
	if _, err := New(conn); err != nil {
		t.Fatal(err)
	}
	created, err := appdb.CreateOrganization(Organization{Name: "Custom", Slug: "custom", PrimaryColor: " #ABCDEF ", SecondaryColor: "#123456", TertiaryColor: "#010203"})
	if err != nil {
		t.Fatal(err)
	}
	if created.PrimaryColor != "#abcdef" || created.SecondaryColor != "#123456" || created.TertiaryColor != "#010203" {
		t.Fatalf("create colors: %+v", created)
	}
	created.PrimaryColor, created.SecondaryColor, created.TertiaryColor = "#112233", "#445566", "#778899"
	updated, err := appdb.UpdateOrganization(created)
	if err != nil {
		t.Fatal(err)
	}
	bySlug, err := appdb.GetOrganizationBySlug("custom")
	if err != nil || bySlug.PrimaryColor != "#112233" || bySlug.SecondaryColor != "#445566" || bySlug.TertiaryColor != "#778899" {
		t.Fatalf("updated colors: %+v, %v", bySlug, err)
	}
	updated.PrimaryColor, updated.SecondaryColor, updated.TertiaryColor = "", "", ""
	preserved, err := appdb.UpdateOrganization(updated)
	if err != nil || preserved.PrimaryColor != bySlug.PrimaryColor || preserved.SecondaryColor != bySlug.SecondaryColor || preserved.TertiaryColor != bySlug.TertiaryColor {
		t.Fatalf("omitted colors: %+v, %v", preserved, err)
	}
	for _, invalid := range []string{"red", "#123", "#gg0000", "#1234567", "var(--red)"} {
		for _, field := range []string{"primary", "secondary", "tertiary"} {
			bad := bySlug
			switch field {
			case "primary":
				bad.PrimaryColor = invalid
			case "secondary":
				bad.SecondaryColor = invalid
			case "tertiary":
				bad.TertiaryColor = invalid
			}
			if _, err := appdb.UpdateOrganization(bad); !errors.Is(err, ErrInvalidOrganizationData) {
				t.Fatalf("%s update accepted %q: %v", field, invalid, err)
			}
			bad.ID = 0
			bad.Name, bad.Slug = "Invalid", "invalid"
			if _, err := appdb.CreateOrganization(bad); !errors.Is(err, ErrInvalidOrganizationData) {
				t.Fatalf("%s create accepted %q: %v", field, invalid, err)
			}
		}
	}
}
