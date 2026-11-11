package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type SponsorPlacementGroup string

const (
	GameWallPlacements SponsorPlacementGroup = "game_wall_sponsors"
	PostVotePlacements SponsorPlacementGroup = "post_vote_sponsors"
)

var ErrInvalidSponsorPlacement = errors.New("invalid placement or sponsor")

var sponsorPlacementGroups = map[SponsorPlacementGroup][]string{
	GameWallPlacements: {"basketball_left", "basketball_right", "playmaker_left", "playmaker_right", "crossover_left", "crossover_right"},
	PostVotePlacements: {"post_vote_left", "post_vote_right"},
}

func ensureSponsorPlacements(db *sql.DB) error {
	for group, placements := range sponsorPlacementGroups {
		_, err := db.Exec(`CREATE TABLE IF NOT EXISTS ` + string(group) + ` (
			organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			placement TEXT NOT NULL CHECK (placement IN ('` + strings.Join(placements, "', '") + `')),
			sponsor_id INTEGER NOT NULL REFERENCES sponsors(id) ON DELETE CASCADE,
			PRIMARY KEY (organization_id, placement)
		)`)
		if err != nil {
			return fmt.Errorf("creating %s: %w", group, err)
		}
	}
	return nil
}

func (db *appdbimpl) GetSponsorPlacements(ctx context.Context, organizationID int, group SponsorPlacementGroup, activeOnly bool) (map[string]int, error) {
	if _, ok := sponsorPlacementGroups[group]; !ok {
		return nil, ErrInvalidSponsorPlacement
	}
	rows, err := db.c.QueryContext(ctx, `
		SELECT w.placement, s.id
		FROM `+string(group)+` w
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

func (db *appdbimpl) SaveSponsorPlacements(ctx context.Context, organizationID int, group SponsorPlacementGroup, assignments map[string]int) error {
	placements, ok := sponsorPlacementGroups[group]
	if !ok {
		return ErrInvalidSponsorPlacement
	}
	for placement, id := range assignments {
		if !validSponsorPlacement(placements, placement) || id < 0 {
			return ErrInvalidSponsorPlacement
		}
	}
	tx, err := db.c.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM `+string(group)+` WHERE organization_id = ?`, organizationID); err != nil {
		return err
	}
	for placement, id := range assignments {
		if id == 0 {
			continue
		}
		result, err := tx.ExecContext(ctx, `
			INSERT INTO `+string(group)+` (organization_id, placement, sponsor_id)
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
			return ErrInvalidSponsorPlacement
		}
	}
	return tx.Commit()
}

func validSponsorPlacement(placements []string, candidate string) bool {
	for _, placement := range placements {
		if placement == candidate {
			return true
		}
	}
	return false
}
