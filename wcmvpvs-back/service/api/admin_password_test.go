package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"testing"

	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/internal/security"
	"github.com/albyma98/wcmvpvotingsystem/wcmvpvs-back/service/database"
	_ "github.com/mattn/go-sqlite3"
)

func newTestRouterWithDB(t *testing.T) *_router {
	t.Helper()
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open(): %v", err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })

	appdb, err := database.New(sqldb)
	if err != nil {
		t.Fatalf("database.New(): %v", err)
	}
	return &_router{db: appdb}
}

func TestVerifyAndMigrateAdminPasswordLegacySHA256(t *testing.T) {
	rt := newTestRouterWithDB(t)
	sum := sha256.Sum256([]byte("legacy-pass"))
	legacyHash := hex.EncodeToString(sum[:])

	id, err := rt.db.CreateAdmin(database.Admin{Username: "legacy", PasswordHash: legacyHash, Role: "staff", OrganizationID: 1})
	if err != nil {
		t.Fatalf("CreateAdmin(): %v", err)
	}
	admin, err := rt.db.GetAdminByID(id)
	if err != nil {
		t.Fatalf("GetAdminByID(): %v", err)
	}

	ok, updatedHash, err := rt.verifyAndMigrateAdminPassword(admin, "legacy-pass")
	if err != nil {
		t.Fatalf("verifyAndMigrateAdminPassword(): %v", err)
	}
	if !ok {
		t.Fatal("expected legacy password to match")
	}
	if !security.IsArgon2idHash(updatedHash) {
		t.Fatalf("expected updated argon2id hash, got %q", updatedHash)
	}

	updatedAdmin, err := rt.db.GetAdminByID(id)
	if err != nil {
		t.Fatalf("GetAdminByID() after migrate: %v", err)
	}
	if !security.IsArgon2idHash(updatedAdmin.PasswordHash) {
		t.Fatalf("expected admin hash to be migrated, got %q", updatedAdmin.PasswordHash)
	}
}

func TestVerifyAndMigrateAdminPasswordArgonNoRehash(t *testing.T) {
	rt := newTestRouterWithDB(t)
	hash, err := security.HashPassword("argon-pass")
	if err != nil {
		t.Fatalf("HashPassword(): %v", err)
	}
	id, err := rt.db.CreateAdmin(database.Admin{Username: "argon", PasswordHash: hash, Role: "staff", OrganizationID: 1})
	if err != nil {
		t.Fatalf("CreateAdmin(): %v", err)
	}
	admin, err := rt.db.GetAdminByID(id)
	if err != nil {
		t.Fatalf("GetAdminByID(): %v", err)
	}

	ok, updatedHash, err := rt.verifyAndMigrateAdminPassword(admin, "argon-pass")
	if err != nil {
		t.Fatalf("verifyAndMigrateAdminPassword(): %v", err)
	}
	if !ok {
		t.Fatal("expected argon2 password to match")
	}
	if updatedHash != "" {
		t.Fatalf("expected no rehash, got %q", updatedHash)
	}
}
