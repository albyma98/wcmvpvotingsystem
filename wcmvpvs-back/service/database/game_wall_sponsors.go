package database

import (
	"context"
	"database/sql"
	"errors"
)

var ErrInvalidGameWallSponsor = errors.New("invalid game wall or sponsor")

func ensureGameWallSponsors(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS game_wall_sponsors (
		organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
		placement TEXT NOT NULL CHECK (placement IN (
			'basketball_left', 'basketball_right', 'playmaker_left', 'playmaker_right',
			'crossover_left', 'crossover_right'
		)),
		sponsor_id INTEGER NOT NULL REFERENCES sponsors(id) ON DELETE CASCADE,
		PRIMARY KEY (organization_id, placement)
	)`)
	return err
}

func validGameWallPlacement(placement string) bool {
	switch placement {
	case "basketball_left", "basketball_right", "playmaker_left", "playmaker_right", "crossover_left", "crossover_right":
		return true
	default:
		return false
	}
}

func (db *appdbimpl) GetGameWallSponsors(ctx context.Context, organizationID int, activeOnly bool) (map[string]int, error) {
	rows, err := db.c.QueryContext(ctx, `
		SELECT w.placement, s.id
		FROM game_wall_sponsors w
		JOIN sponsors s ON s.id = w.sponsor_id AND s.organization_id = w.organization_id
		WHERE w.organization_id = ? AND (? = 0 OR s.is_active = 1)`, organizationID, boolToInt(activeOnly))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	assignments := make(map[string]int)
	for rows.Next() {
		var placement string
		var id int
		if err := rows.Scan(&placement, &id); err != nil {
			return nil, err
		}
		assignments[placement] = id
	}
	return assignments, rows.Err()
}

func (db *appdbimpl) SaveGameWallSponsors(ctx context.Context, organizationID int, assignments map[string]int) error {
	for placement, id := range assignments {
		if !validGameWallPlacement(placement) || id < 0 {
			return ErrInvalidGameWallSponsor
		}
	}
	tx, err := db.c.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM game_wall_sponsors WHERE organization_id = ?`, organizationID); err != nil {
		return err
	}
	for placement, id := range assignments {
		if id == 0 {
			continue
		}
		result, err := tx.ExecContext(ctx, `
			INSERT INTO game_wall_sponsors (organization_id, placement, sponsor_id)
			SELECT organization_id, ?, id FROM sponsors WHERE organization_id = ? AND id = ?`,
			placement, organizationID, id)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return ErrInvalidGameWallSponsor
		}
	}
	return tx.Commit()
}
