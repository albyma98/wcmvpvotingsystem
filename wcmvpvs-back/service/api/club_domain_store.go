package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

var errClubDomainConflict = errors.New("domain already registered")

const clubDomainTimestamp = "2006-01-02T15:04:05.000000000Z"

type ClubDomain struct {
	ID                int64  `json:"id"`
	SiteID            int64  `json:"siteId"`
	Hostname          string `json:"hostname"`
	Primary           bool   `json:"primary"`
	Status            string `json:"status"`
	VerificationName  string `json:"verificationName"`
	VerificationValue string `json:"verificationValue"`
	Message           string `json:"message"`
	CheckedAt         string `json:"checkedAt"`
	VerifiedAt        string `json:"verifiedAt"`
}

func (s *Store) ensureClubDomainTables() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS club_domains (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		site_id INTEGER NOT NULL,
		hostname TEXT NOT NULL UNIQUE COLLATE NOCASE,
		is_primary INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'pending',
		verification_name TEXT NOT NULL,
		verification_value TEXT NOT NULL,
		message TEXT NOT NULL DEFAULT 'In attesa della verifica DNS.',
		checked_at TEXT NOT NULL DEFAULT '',
		verified_at TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_club_domains_site ON club_domains(site_id);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_club_domains_primary ON club_domains(site_id) WHERE is_primary=1;`)
	return err
}

const clubDomainColumns = `id,site_id,hostname,is_primary,status,verification_name,verification_value,message,checked_at,verified_at`

type clubDomainScanner interface{ Scan(...interface{}) error }

func scanClubDomain(row clubDomainScanner) (ClubDomain, error) {
	var d ClubDomain
	err := row.Scan(&d.ID, &d.SiteID, &d.Hostname, &d.Primary, &d.Status, &d.VerificationName, &d.VerificationValue, &d.Message, &d.CheckedAt, &d.VerifiedAt)
	return d, err
}

func (s *Store) ListClubDomains(ctx context.Context, siteID int64) ([]ClubDomain, error) {
	query := `SELECT ` + clubDomainColumns + ` FROM club_domains`
	args := []interface{}{}
	if siteID > 0 {
		query += ` WHERE site_id=?`
		args = append(args, siteID)
	}
	query += ` ORDER BY checked_at,id`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ClubDomain{}
	for rows.Next() {
		d, e := scanClubDomain(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, d)
	}
	return items, rows.Err()
}

func (s *Store) AddClubDomains(ctx context.Context, siteID int64, hostname string, includeWWW bool) error {
	token := make([]byte, 24)
	if _, err := rand.Read(token); err != nil {
		return err
	}
	hosts := []string{hostname}
	if includeWWW {
		hosts = append(hosts, "www."+hostname)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM club_sites WHERE id=?`, siteID).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return errClubNotFound
	}
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM club_domains WHERE site_id=?`, siteID).Scan(&count); err != nil {
		return err
	}
	if count+len(hosts) > 10 {
		return errors.New("domain_limit")
	}
	for i, host := range hosts {
		var exists int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM club_domains WHERE hostname=?`, host).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			return errClubDomainConflict
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO club_domains(site_id,hostname,is_primary,verification_name,verification_value) VALUES(?,?,?,?,?)`, siteID, host, count == 0 && i == 0, "_clubhub."+hostname, "clubhub-verification="+hex.EncodeToString(token))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) UpdateClubDomainCheck(ctx context.Context, d ClubDomain, valid bool, message string) error {
	now := time.Now().UTC().Format(clubDomainTimestamp)
	status := "pending"
	if valid {
		status = "active"
	}
	_, err := s.db.ExecContext(ctx, `UPDATE club_domains SET status=?,message=?,checked_at=?,verified_at=CASE WHEN ? THEN ? ELSE verified_at END WHERE id=? AND verification_value=?`, status, message, now, valid, now, d.ID, d.VerificationValue)
	return err
}

func (s *Store) DeleteClubDomain(ctx context.Context, siteID, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `DELETE FROM club_domains WHERE id=? AND site_id=?`, id, siteID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errClubNotFound
	}
	_, err = tx.ExecContext(ctx, `UPDATE club_domains SET is_primary=1 WHERE id=(SELECT id FROM club_domains WHERE site_id=? ORDER BY (status='active') DESC,id LIMIT 1) AND NOT EXISTS(SELECT 1 FROM club_domains WHERE site_id=? AND is_primary=1)`, siteID, siteID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SetPrimaryClubDomain(ctx context.Context, siteID, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM club_domains WHERE site_id=? AND id=?`, siteID, id).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return errClubNotFound
	}
	if _, err = tx.ExecContext(ctx, `UPDATE club_domains SET is_primary=0 WHERE site_id=?`, siteID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE club_domains SET is_primary=1 WHERE site_id=? AND id=?`, siteID, id); err != nil {
		return err
	}
	return tx.Commit()
}

// This indexed lookup is used during TLS handshakes: it must not perform DNS I/O.
func (s *Store) ResolveClubDomain(ctx context.Context, hostname string) (slug, primary string, err error) {
	cutoff := time.Now().Add(-24 * time.Hour).UTC().Format(clubDomainTimestamp)
	err = s.db.QueryRowContext(ctx, `SELECT c.slug,COALESCE((SELECT p.hostname FROM club_domains p WHERE p.site_id=c.id AND p.is_primary=1 AND p.status='active' AND p.verified_at>=?),d.hostname)
		FROM club_domains d JOIN club_sites c ON c.id=d.site_id
		WHERE d.hostname=? AND d.status='active' AND d.verified_at>=? AND c.active=1`, cutoff, hostname, cutoff).Scan(&slug, &primary)
	return
}

func clubDomainNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, errClubNotFound)
}
