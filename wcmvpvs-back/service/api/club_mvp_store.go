package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
)

//go:embed club_mvp_migrations/001_mvp_light.sql
var clubMVPSchema string

var (
	errClubMVPClosed = errors.New("mvp_closed")
	errClubMVPPlayer = errors.New("invalid_player")
	errClubMVPLocked = errors.New("callups_locked")
	errClubMVPEmpty  = errors.New("no_callups")
)

type clubMVPPlayer struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	JerseyNumber int    `json:"jerseyNumber"`
	Role         string `json:"role"`
}

type clubMVPPublic struct {
	Club    clubMVPBrand    `json:"club"`
	Match   ClubMatch       `json:"match"`
	Team    clubMVPDivision `json:"team"`
	Players []clubMVPPlayer `json:"players"`
	Voted   *clubMVPPlayer  `json:"voted,omitempty"`
}

type clubMVPBrand struct {
	ID             int64  `json:"-"`
	Name           string `json:"name"`
	LogoURL        string `json:"logoUrl"`
	PrimaryColor   string `json:"primaryColor"`
	SecondaryColor string `json:"secondaryColor"`
	TitleFont      string `json:"titleFont"`
	BodyFont       string `json:"bodyFont"`
}

type clubMVPDivision struct {
	Name    string `json:"name"`
	LogoURL string `json:"logoUrl"`
	Active  bool   `json:"-"`
}

type clubMVPRank struct {
	clubMVPPlayer
	Votes      int     `json:"votes"`
	Percentage float64 `json:"percentage"`
	Leader     bool    `json:"leader"`
	Eligible   bool    `json:"eligible"`
}

type clubMVPAdmin struct {
	Match        ClubMatch       `json:"match"`
	Team         clubMVPDivision `json:"team"`
	Roster       []ClubPlayer    `json:"roster"`
	Callups      []clubMVPRank   `json:"callups"`
	Enabled      bool            `json:"enabled"`
	Open         bool            `json:"open"`
	QRTarget     bool            `json:"qrTarget"`
	OpenedAt     string          `json:"openedAt"`
	ClosedAt     string          `json:"closedAt"`
	FinalizedAt  string          `json:"finalizedAt"`
	Final        bool            `json:"final"`
	CanOpen      bool            `json:"canOpen"`
	TotalVotes   int             `json:"totalVotes"`
	PlayersVoted int             `json:"playersVoted"`
}

type clubMVPChange struct {
	PlayerIDs *[]int64 `json:"playerIds"`
	Enabled   *bool    `json:"enabled"`
	QRTarget  *bool    `json:"qrTarget"`
}

func (s *Store) ensureClubMVPTables() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(clubMVPSchema); err != nil {
		return fmt.Errorf("club MVP Light migration: %w", err)
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO club_mvp_cookie_key(id, secret) VALUES (1, ?)`, secret[:]); err != nil {
		return err
	}
	return tx.Commit()
}

func clubMVPMatch(ctx context.Context, tx *sql.Tx, siteID, matchID int64) (ClubMatch, clubMVPDivision, error) {
	match, err := scanClubMatch(tx.QueryRowContext(ctx, clubMatchSelect+" AND id=?", siteID, matchID))
	var team clubMVPDivision
	if err == nil {
		err = tx.QueryRowContext(ctx, `SELECT name, logo_url, active FROM club_teams WHERE id=? AND site_id=?`,
			match.TeamID, siteID).Scan(&team.Name, &team.LogoURL, &team.Active)
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = errClubNotFound
	}
	return match, team, err
}

// A call-up is eligible only while its roster ID belongs to the match's division.
// Its displayed details come from the snapshot, including for historical votes.
const clubMVPRosterJoin = `p.id=c.player_id AND p.site_id=m.site_id AND p.team_id=m.team_id`
const clubMVPOpenMatches = `
	FROM club_matches m
	JOIN club_mvp_matches cfg ON cfg.match_id=m.id AND cfg.site_id=m.site_id
	JOIN club_sites s ON s.id=m.site_id AND s.active=1
	JOIN club_teams t ON t.id=m.team_id AND t.site_id=m.site_id AND t.active=1
	WHERE m.site_id=? AND m.status='live' AND cfg.enabled=1 AND cfg.finalized_at IS NULL
	AND EXISTS (SELECT 1 FROM club_mvp_callups c JOIN club_players p ON ` + clubMVPRosterJoin + ` WHERE c.match_id=m.id)`

func clubMVPTarget(ctx context.Context, tx *sql.Tx, siteID, matchID int64) (int64, error) {
	if matchID != 0 {
		var target int64
		err := tx.QueryRowContext(ctx, `SELECT m.id`+clubMVPOpenMatches+` AND m.id=?`, siteID, matchID).Scan(&target)
		if errors.Is(err, sql.ErrNoRows) {
			err = errClubMVPClosed
		}
		return target, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT m.id,cfg.qr_target`+clubMVPOpenMatches+` ORDER BY cfg.qr_target DESC LIMIT 2`, siteID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var target int64
	var preferred bool
	if rows.Next() {
		if err := rows.Scan(&target, &preferred); err != nil {
			return 0, err
		}
		if rows.Next() && !preferred {
			target = 0
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if target == 0 {
		return 0, errClubMVPClosed
	}
	return target, nil
}

const clubMVPPlayerColumns = `c.player_id,c.first_name,c.last_name,c.jersey_number,c.role`

func clubMVPVotedPlayer(ctx context.Context, tx *sql.Tx, matchID int64, voterHash []byte) (*clubMVPPlayer, error) {
	if len(voterHash) == 0 {
		return nil, nil
	}
	var p clubMVPPlayer
	err := tx.QueryRowContext(ctx, `SELECT `+clubMVPPlayerColumns+`
		FROM club_mvp_votes v JOIN club_mvp_callups c ON c.match_id=v.match_id AND c.player_id=v.player_id
		WHERE v.match_id=? AND v.voter_hash=?`, matchID, voterHash).
		Scan(&p.ID, &p.FirstName, &p.LastName, &p.JerseyNumber, &p.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

func (s *Store) clubMVPPublic(ctx context.Context, site clubMVPBrand, matchID int64, voterHash []byte) (clubMVPPublic, error) {
	out := clubMVPPublic{Club: site, Players: []clubMVPPlayer{}}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	id, err := clubMVPTarget(ctx, tx, site.ID, matchID)
	if err != nil {
		return out, err
	}
	out.Match, out.Team, err = clubMVPMatch(ctx, tx, site.ID, id)
	if err != nil {
		return out, err
	}
	out.Voted, err = clubMVPVotedPlayer(ctx, tx, id, voterHash)
	if err != nil {
		return out, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT `+clubMVPPlayerColumns+`
		FROM club_mvp_callups c JOIN club_matches m ON m.id=c.match_id
		JOIN club_players p ON `+clubMVPRosterJoin+`
		WHERE c.match_id=? ORDER BY c.position,c.jersey_number,c.player_id`, id)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var p clubMVPPlayer
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.JerseyNumber, &p.Role); err != nil {
			return out, err
		}
		out.Players = append(out.Players, p)
	}
	return out, rows.Err()
}

func (s *Store) castClubMVPVote(ctx context.Context, siteID, matchID, playerID int64, voterHash []byte) (*clubMVPPlayer, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	// Start with a write. Validation, the unique constraint and the match
	// completion trigger then share SQLite's writer lock.
	res, err := tx.ExecContext(ctx, `INSERT INTO club_mvp_votes(match_id,player_id,voter_hash)
		SELECT m.id,?,?`+clubMVPOpenMatches+` AND m.id=? AND EXISTS (
			SELECT 1 FROM club_mvp_callups c JOIN club_players p ON `+clubMVPRosterJoin+`
			WHERE c.match_id=m.id AND c.player_id=?
		) ON CONFLICT (match_id,voter_hash) DO NOTHING`, playerID, voterHash, siteID, matchID, playerID)
	if err != nil {
		return nil, false, err
	}
	inserted, err := res.RowsAffected()
	if err != nil {
		return nil, false, err
	}
	if inserted == 0 {
		if _, err := clubMVPTarget(ctx, tx, siteID, matchID); err != nil {
			return nil, false, err
		}
	}
	player, err := clubMVPVotedPlayer(ctx, tx, matchID, voterHash)
	if err != nil {
		return nil, false, err
	}
	if player == nil {
		return nil, false, errClubMVPPlayer
	}
	return player, inserted == 0, tx.Commit()
}

func (s *Store) clubMVPAdmin(ctx context.Context, siteID, matchID int64) (clubMVPAdmin, error) {
	out := clubMVPAdmin{Callups: []clubMVPRank{}}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	out.Match, out.Team, err = clubMVPMatch(ctx, tx, siteID, matchID)
	if err != nil {
		return out, err
	}
	out.Roster, err = listClubPlayers(ctx, tx, siteID, out.Match.TeamID, false)
	if err != nil {
		return out, err
	}
	err = tx.QueryRowContext(ctx, `SELECT enabled,qr_target,COALESCE(opened_at,''),COALESCE(closed_at,''),COALESCE(finalized_at,'')
		FROM club_mvp_matches WHERE site_id=? AND match_id=?`, siteID, matchID).
		Scan(&out.Enabled, &out.QRTarget, &out.OpenedAt, &out.ClosedAt, &out.FinalizedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return out, err
	}
	out.Final = out.Match.Status == "completed" || out.FinalizedAt != ""
	var siteActive bool
	if err := tx.QueryRowContext(ctx, `SELECT active FROM club_sites WHERE id=?`, siteID).Scan(&siteActive); err != nil {
		return out, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT `+clubMVPPlayerColumns+`,COUNT(v.id),p.id IS NOT NULL
		FROM club_mvp_callups c JOIN club_matches m ON m.id=c.match_id
		LEFT JOIN club_players p ON `+clubMVPRosterJoin+`
		LEFT JOIN club_mvp_votes v ON v.match_id=c.match_id AND v.player_id=c.player_id
		WHERE c.match_id=? GROUP BY c.match_id,c.player_id
		ORDER BY COUNT(v.id) DESC,c.position,c.jersey_number,c.player_id`, matchID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	hasEligible := false
	for rows.Next() {
		var rank clubMVPRank
		if err := rows.Scan(&rank.ID, &rank.FirstName, &rank.LastName, &rank.JerseyNumber, &rank.Role, &rank.Votes, &rank.Eligible); err != nil {
			return out, err
		}
		out.TotalVotes += rank.Votes
		if rank.Votes > 0 {
			out.PlayersVoted++
		}
		hasEligible = hasEligible || rank.Eligible
		out.Callups = append(out.Callups, rank)
	}
	for i := range out.Callups {
		if out.TotalVotes > 0 {
			out.Callups[i].Percentage = float64(out.Callups[i].Votes) * 100 / float64(out.TotalVotes)
			out.Callups[i].Leader = out.Callups[i].Votes == out.Callups[0].Votes
		}
	}
	out.CanOpen = !out.Final && out.Match.Status == "live" && siteActive && out.Team.Active && hasEligible
	out.Open = out.CanOpen && out.Enabled
	return out, rows.Err()
}

func (s *Store) updateClubMVP(ctx context.Context, siteID, matchID int64, change clubMVPChange) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO club_mvp_matches(match_id,site_id)
		SELECT id,site_id FROM club_matches WHERE site_id=? AND id=? ON CONFLICT (match_id) DO NOTHING`, siteID, matchID); err != nil {
		return err
	}
	match, team, err := clubMVPMatch(ctx, tx, siteID, matchID)
	if err != nil {
		return err
	}
	var final, siteActive bool
	err = tx.QueryRowContext(ctx, `SELECT cfg.finalized_at IS NOT NULL,s.active
		FROM club_mvp_matches cfg JOIN club_sites s ON s.id=cfg.site_id WHERE cfg.match_id=?`, matchID).Scan(&final, &siteActive)
	if err != nil {
		return err
	}
	if match.Status == "completed" || final {
		return errClubMVPClosed
	}
	if change.PlayerIDs != nil {
		if err := saveClubMVPCallups(ctx, tx, siteID, match, *change.PlayerIDs); err != nil {
			return err
		}
	}
	if change.Enabled != nil {
		if *change.Enabled {
			if match.Status != "live" || !siteActive || !team.Active {
				return errClubMVPClosed
			}
			if _, err := tx.ExecContext(ctx, `UPDATE club_mvp_matches SET enabled=1,
				opened_at=CASE WHEN enabled=0 THEN CURRENT_TIMESTAMP ELSE opened_at END,closed_at=NULL WHERE match_id=?`, matchID); err != nil {
				return err
			}
			if _, err := clubMVPTarget(ctx, tx, siteID, matchID); errors.Is(err, errClubMVPClosed) {
				return errClubMVPEmpty
			} else if err != nil {
				return err
			}
		} else if _, err := tx.ExecContext(ctx, `UPDATE club_mvp_matches SET
			closed_at=CASE WHEN enabled=1 THEN CURRENT_TIMESTAMP ELSE closed_at END,enabled=0,qr_target=0 WHERE match_id=?`, matchID); err != nil {
			return err
		}
	}
	if change.QRTarget != nil {
		if *change.QRTarget {
			if _, err := clubMVPTarget(ctx, tx, siteID, matchID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE club_mvp_matches SET qr_target=0 WHERE site_id=? AND qr_target=1`, siteID); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE club_mvp_matches SET qr_target=? WHERE match_id=?`, *change.QRTarget, matchID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func saveClubMVPCallups(ctx context.Context, tx *sql.Tx, siteID int64, match ClubMatch, ids []int64) error {
	var hasVotes bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM club_mvp_votes WHERE match_id=?)`, match.ID).Scan(&hasVotes); err != nil {
		return err
	}
	if hasVotes {
		return errClubMVPLocked
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM club_mvp_callups WHERE match_id=?`, match.ID); err != nil {
		return err
	}
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return errClubMVPPlayer
		}
		seen[id] = true
		res, err := tx.ExecContext(ctx, `INSERT INTO club_mvp_callups(match_id,player_id,first_name,last_name,jersey_number,role,position)
			SELECT ?,id,first_name,last_name,jersey_number,role,position FROM club_players WHERE id=? AND site_id=? AND team_id=?`,
			match.ID, id, siteID, match.TeamID)
		if err != nil {
			return err
		}
		if count, err := res.RowsAffected(); err != nil {
			return err
		} else if count != 1 {
			return errClubMVPPlayer
		}
	}
	if len(ids) == 0 {
		_, err := tx.ExecContext(ctx, `UPDATE club_mvp_matches SET
			closed_at=CASE WHEN enabled=1 THEN CURRENT_TIMESTAMP ELSE closed_at END,enabled=0,qr_target=0 WHERE match_id=?`, match.ID)
		return err
	}
	return nil
}
