package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const VotingDrawRandom = "random"
const VotingDrawLeaderboard = "leaderboard"

var ErrInvalidVotingSettings = errors.New("invalid voting settings")
var ErrEventVotingClosed = errors.New("event voting closed")
var ErrVotingClosureLocked = errors.New("automatic voting results are locked")
var votingTemplateSID = regexp.MustCompile(`^HX[0-9a-fA-F]{32}$`)
var votingRecipientPhone = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)

type VotingRecipient struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Enabled bool   `json:"enabled"`
}

type VotingSettings struct {
	TemplateSID string            `json:"template_sid"`
	DrawMode    string            `json:"draw_mode"`
	Recipients  []VotingRecipient `json:"recipients"`
}

type VotingClosurePrize struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Position   int    `json:"position"`
	VoteID     int    `json:"vote_id"`
	FanID      int    `json:"fan_id"`
	Nickname   string `json:"nickname"`
	TicketCode string `json:"ticket_code"`
}

type VotingClosure struct {
	EventID    int                  `json:"event_id"`
	DrawMode   string               `json:"draw_mode"`
	MVPName    string               `json:"mvp_name"`
	MVPVotes   int                  `json:"mvp_votes"`
	TotalVotes int                  `json:"total_votes"`
	Percentage string               `json:"percentage"`
	Prizes     []VotingClosurePrize `json:"prizes"`
	Warnings   []string             `json:"warnings"`
	CreatedAt  string               `json:"created_at"`
}

// TemplateVariables always fills all eight placeholders, including events with
// fewer than three prizes or too few eligible voters. No sample/default values
// from the provider's template are allowed to stand in for actual results.
func (c VotingClosure) TemplateVariables() map[string]string {
	variables := map[string]string{"1": c.MVPName, "2": c.Percentage}
	for i := 0; i < 3; i++ {
		name, code := "Premio non previsto", "-"
		if i < len(c.Prizes) {
			name, code = c.Prizes[i].Name, c.Prizes[i].TicketCode
			if code == "" {
				code = "Non assegnato"
			}
		}
		variables[strconv.Itoa(3+i*2)] = singleLineVotingValue(name)
		variables[strconv.Itoa(4+i*2)] = code
	}
	variables["1"] = singleLineVotingValue(variables["1"])
	return variables
}

func singleLineVotingValue(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

type VotingNotification struct {
	ID             int               `json:"id"`
	EventID        int               `json:"-"`
	OrganizationID int               `json:"-"`
	Kind           string            `json:"-"`
	PrizeID        int               `json:"-"`
	Name           string            `json:"name"`
	Phone          string            `json:"phone"`
	TemplateSID    string            `json:"-"`
	Variables      map[string]string `json:"-"`
	Status         string            `json:"status"`
	ProviderSID    string            `json:"-"`
	ProviderStatus string            `json:"provider_status,omitempty"`
	Error          string            `json:"error,omitempty"`
}

func ensureVotingClosureTables(db *sql.DB) error {
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS organization_voting_settings (
			organization_id INTEGER PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
			template_sid TEXT NOT NULL DEFAULT '', draw_mode TEXT NOT NULL DEFAULT 'random',
			recipients_json TEXT NOT NULL DEFAULT '[]');`,
		`CREATE TABLE IF NOT EXISTS event_voting_closures (
			event_id INTEGER PRIMARY KEY REFERENCES events(id) ON DELETE CASCADE,
			organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			summary_json TEXT NOT NULL);`,
		`CREATE TABLE IF NOT EXISTS voting_notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			kind TEXT NOT NULL, prize_id INTEGER NOT NULL DEFAULT 0,
			name TEXT NOT NULL DEFAULT '', phone TEXT NOT NULL, template_sid TEXT NOT NULL DEFAULT '',
			variables_json TEXT NOT NULL DEFAULT '{}', status TEXT NOT NULL DEFAULT 'pending',
			provider_sid TEXT NOT NULL DEFAULT '', provider_status TEXT NOT NULL DEFAULT '',
			error TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(event_id,kind,prize_id,phone));`,
		`CREATE INDEX IF NOT EXISTS idx_voting_notifications_pending ON voting_notifications(status,id);`,
	} {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("voting closure schema: %w", err)
		}
	}
	return nil
}

type votingRowQuerier interface {
	QueryRow(string, ...interface{}) *sql.Row
}

func loadVotingSettings(q votingRowQuerier, organizationID int) (VotingSettings, error) {
	settings := VotingSettings{DrawMode: VotingDrawRandom, Recipients: []VotingRecipient{}}
	var recipients string
	err := q.QueryRow(`SELECT template_sid,draw_mode,recipients_json FROM organization_voting_settings WHERE organization_id=?`, organizationID).Scan(&settings.TemplateSID, &settings.DrawMode, &recipients)
	if errors.Is(err, sql.ErrNoRows) {
		return settings, nil
	}
	if err == nil {
		err = json.Unmarshal([]byte(recipients), &settings.Recipients)
	}
	return settings, err
}

func (db *appdbimpl) GetVotingSettings(organizationID int) (VotingSettings, error) {
	return loadVotingSettings(db.c, organizationID)
}

func (db *appdbimpl) SaveVotingSettings(organizationID int, settings VotingSettings) (VotingSettings, error) {
	settings.TemplateSID = strings.TrimSpace(settings.TemplateSID)
	if organizationID <= 0 || (settings.DrawMode != VotingDrawRandom && settings.DrawMode != VotingDrawLeaderboard) ||
		(settings.TemplateSID != "" && !votingTemplateSID.MatchString(settings.TemplateSID)) || len(settings.Recipients) > 20 {
		return settings, ErrInvalidVotingSettings
	}
	seen := map[string]bool{}
	if settings.Recipients == nil {
		settings.Recipients = []VotingRecipient{}
	}
	for i := range settings.Recipients {
		r := &settings.Recipients[i]
		r.Name = strings.TrimSpace(r.Name)
		// Accept pasted international numbers with ordinary visual separators.
		r.Phone = strings.Map(func(c rune) rune {
			if c == ' ' || c == '-' || c == '(' || c == ')' {
				return -1
			}
			return c
		}, strings.TrimSpace(r.Phone))
		if len(r.Name) > 100 || !votingRecipientPhone.MatchString(r.Phone) || seen[r.Phone] || (r.Enabled && settings.TemplateSID == "") {
			return settings, ErrInvalidVotingSettings
		}
		seen[r.Phone] = true
	}
	recipients, err := json.Marshal(settings.Recipients)
	if err != nil {
		return settings, err
	}
	_, err = db.c.Exec(`INSERT INTO organization_voting_settings(organization_id,template_sid,draw_mode,recipients_json) VALUES(?,?,?,?)
		ON CONFLICT(organization_id) DO UPDATE SET template_sid=excluded.template_sid,draw_mode=excluded.draw_mode,recipients_json=excluded.recipients_json`,
		organizationID, settings.TemplateSID, settings.DrawMode, string(recipients))
	return settings, err
}

type votingCandidate struct {
	VoteID, FanID, Points int
	Code, Nickname, Phone string
}

func lockVotingPrizesForManualChange(tx *sql.Tx, eventID int) error {
	if _, err := tx.Exec(`UPDATE events SET votes_closed=votes_closed WHERE id=?`, eventID); err != nil {
		return err
	}
	var closed int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM event_voting_closures WHERE event_id=?`, eventID).Scan(&closed); err != nil {
		return err
	}
	if closed > 0 {
		return ErrVotingClosureLocked
	}
	return nil
}

// This uses the lottery's four-digit tickets and verified fan-session linkage.
// The latest eligible session owns each ticket, as in AssignPrizeWinner.
func votingCandidates(tx *sql.Tx, eventID, organizationID int) ([]votingCandidate, error) {
	rows, err := tx.Query(`SELECT v.id,TRIM(v.ticket_code),p.id,p.nickname,p.phone_e164,COALESCE(a.points,w.coins,0)
		FROM votes v JOIN fan_profiles p ON p.id=(
			SELECT fs.fan_id FROM fan_sessions fs JOIN fan_profiles fp ON fp.id=fs.fan_id
			WHERE fs.device_id=v.device_id AND fp.organization_id=?
			AND TRIM(IFNULL(fp.phone_verified_at,''))<>'' AND TRIM(IFNULL(fp.phone_e164,''))<>''
			ORDER BY fs.last_seen_at DESC,fs.created_at DESC,fs.token DESC LIMIT 1)
		LEFT JOIN fan_wallets w ON w.fan_id=p.id
		LEFT JOIN fan_economy_accounts a ON a.organization_id=p.organization_id AND a.owner_key='fan:'||p.id
		WHERE v.event_id=? AND LENGTH(TRIM(v.ticket_code))=4 AND TRIM(v.ticket_code) GLOB '[0-9][0-9][0-9][0-9]'
		ORDER BY v.created_at ASC,v.id ASC`, organizationID, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var candidates []votingCandidate
	for rows.Next() {
		var c votingCandidate
		if err := rows.Scan(&c.VoteID, &c.Code, &c.FanID, &c.Nickname, &c.Phone, &c.Points); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}
	return candidates, rows.Err()
}

// CloseEventVotingAndDraw freezes results, assigns prizes, and enqueues messages
// in one transaction. The first write serializes concurrent closing requests.
// A saved closure is immutable: reopening/reclosing never draws or sends twice.
func (db *appdbimpl) CloseEventVotingAndDraw(eventID, organizationID int) (VotingClosure, error) {
	var summary VotingClosure
	tx, err := db.c.Begin()
	if err != nil {
		return summary, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE events SET votes_closed=1 WHERE id=? AND organization_id=? AND is_active=1 AND is_concluded=0`, eventID, organizationID)
	if err != nil {
		return summary, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return summary, err
	} else if n == 0 {
		return summary, sql.ErrNoRows
	}
	var saved string
	err = tx.QueryRow(`SELECT summary_json FROM event_voting_closures WHERE event_id=? AND organization_id=?`, eventID, organizationID).Scan(&saved)
	if err == nil {
		if err := json.Unmarshal([]byte(saved), &summary); err != nil {
			return summary, err
		}
		return summary, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return summary, err
	}
	settings, err := loadVotingSettings(tx, organizationID)
	if err != nil {
		return summary, err
	}
	summary = VotingClosure{EventID: eventID, DrawMode: settings.DrawMode, MVPName: "Nessun voto", Percentage: "0,00",
		Prizes: []VotingClosurePrize{}, Warnings: []string{}, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	if err := tx.QueryRow(`SELECT COUNT(*) FROM votes WHERE event_id=?`, eventID).Scan(&summary.TotalVotes); err != nil {
		return summary, err
	}
	var firstName, lastName string
	err = tx.QueryRow(`SELECT p.first_name,p.last_name,COUNT(*) AS n FROM votes v JOIN players p ON p.id=v.player_id
		WHERE v.event_id=? GROUP BY p.id ORDER BY n DESC,MAX(v.created_at) ASC,p.last_name COLLATE NOCASE,p.first_name COLLATE NOCASE,p.id LIMIT 1`, eventID).
		Scan(&firstName, &lastName, &summary.MVPVotes)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return summary, err
	}
	if summary.TotalVotes > 0 {
		summary.MVPName = strings.TrimSpace(firstName + " " + lastName)
		summary.Percentage = strings.ReplaceAll(fmt.Sprintf("%.2f", float64(summary.MVPVotes)*100/float64(summary.TotalVotes)), ".", ",")
	} else {
		summary.Warnings = append(summary.Warnings, "Nessun voto registrato per questo evento.")
	}
	rows, err := tx.Query(`SELECT p.id,p.name,p.position,IFNULL(p.winner_vote_id,0),IFNULL(p.winner_user_id,0),
		IFNULL(p.winner_lottery_code,''),IFNULL(fp.nickname,'') FROM event_prizes p LEFT JOIN fan_profiles fp ON fp.id=p.winner_user_id
		WHERE p.event_id=? ORDER BY p.position,p.id`, eventID)
	if err != nil {
		return summary, err
	}
	for rows.Next() {
		var p VotingClosurePrize
		if err := rows.Scan(&p.ID, &p.Name, &p.Position, &p.VoteID, &p.FanID, &p.TicketCode, &p.Nickname); err != nil {
			rows.Close()
			return summary, err
		}
		summary.Prizes = append(summary.Prizes, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return summary, err
	}
	candidates, err := votingCandidates(tx, eventID, organizationID)
	if err != nil {
		return summary, err
	}
	usedVotes, usedFans := map[int]bool{}, map[int]bool{}
	for _, p := range summary.Prizes {
		if p.VoteID > 0 {
			usedVotes[p.VoteID], usedFans[p.FanID] = true, true
		}
	}
	if len(usedVotes) > 0 {
		summary.Warnings = append(summary.Warnings, "I premi già assegnati nella lotteria sono stati mantenuti.")
	}
	if settings.DrawMode == VotingDrawLeaderboard {
		sort.SliceStable(candidates, func(i, j int) bool {
			if candidates[i].Points != candidates[j].Points {
				return candidates[i].Points > candidates[j].Points
			}
			if candidates[i].FanID != candidates[j].FanID {
				return candidates[i].FanID < candidates[j].FanID
			}
			return candidates[i].VoteID < candidates[j].VoteID
		})
	}
	for i := range summary.Prizes {
		p := &summary.Prizes[i]
		if p.VoteID > 0 {
			continue
		} // Keep prizes already assigned by the existing lottery.
		if settings.DrawMode == VotingDrawLeaderboard && i >= 3 {
			summary.Warnings = append(summary.Warnings, "Premio \""+p.Name+"\" non assegnato: la modalità classifica prevede i primi tre premi.")
			continue
		}
		available := make([]votingCandidate, 0, len(candidates))
		for _, c := range candidates {
			if !usedVotes[c.VoteID] && (settings.DrawMode != VotingDrawLeaderboard || !usedFans[c.FanID]) {
				available = append(available, c)
			}
		}
		if len(available) == 0 {
			summary.Warnings = append(summary.Warnings, "Premio \""+p.Name+"\" non assegnato: tifosi votanti idonei insufficienti.")
			continue
		}
		index := 0
		if settings.DrawMode == VotingDrawRandom {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(available))))
			if err != nil {
				return summary, err
			}
			index = int(n.Int64())
		}
		winner := available[index]
		if _, err := tx.Exec(`UPDATE event_prizes SET winner_vote_id=?,winner_user_id=?,winner_lottery_code=?,winner_status='assigned',
			winner_assigned_at=CURRENT_TIMESTAMP,winner_notified_at=NULL,winner_sms_sid=NULL WHERE id=? AND event_id=? AND winner_vote_id IS NULL`,
			winner.VoteID, winner.FanID, winner.Code, p.ID, eventID); err != nil {
			return summary, err
		}
		p.VoteID, p.FanID, p.TicketCode, p.Nickname = winner.VoteID, winner.FanID, winner.Code, winner.Nickname
		usedVotes[winner.VoteID], usedFans[winner.FanID] = true, true
		if err := enqueueVotingNotification(tx, eventID, organizationID, "winner", p.ID, winner.Nickname, winner.Phone, "", nil); err != nil {
			return summary, err
		}
	}
	if len(summary.Prizes) > 3 {
		summary.Warnings = append(summary.Warnings, "Il template WhatsApp riporta solo i primi tre premi.")
	}
	data, err := json.Marshal(summary)
	if err != nil {
		return summary, err
	}
	if _, err := tx.Exec(`INSERT INTO event_voting_closures(event_id,organization_id,summary_json) VALUES(?,?,?)`, eventID, organizationID, string(data)); err != nil {
		return summary, err
	}
	for _, r := range settings.Recipients {
		if r.Enabled {
			if err := enqueueVotingNotification(tx, eventID, organizationID, "organizer", 0, r.Name, r.Phone, settings.TemplateSID, summary.TemplateVariables()); err != nil {
				return summary, err
			}
		}
	}
	return summary, tx.Commit()
}

func enqueueVotingNotification(tx *sql.Tx, eventID, organizationID int, kind string, prizeID int, name, phone, template string, variables map[string]string) error {
	if variables == nil {
		variables = map[string]string{}
	}
	data, err := json.Marshal(variables)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO voting_notifications(event_id,organization_id,kind,prize_id,name,phone,template_sid,variables_json) VALUES(?,?,?,?,?,?,?,?)`,
		eventID, organizationID, kind, prizeID, name, phone, template, string(data))
	return err
}

func (db *appdbimpl) GetVotingClosure(eventID, organizationID int) (VotingClosure, []VotingNotification, error) {
	var summary VotingClosure
	var data string
	err := db.c.QueryRow(`SELECT summary_json FROM event_voting_closures WHERE event_id=? AND organization_id=?`, eventID, organizationID).Scan(&data)
	if err != nil {
		return summary, nil, err
	}
	if err = json.Unmarshal([]byte(data), &summary); err != nil {
		return summary, nil, err
	}
	rows, err := db.c.Query(`SELECT id,name,phone,status,provider_status,error FROM voting_notifications WHERE event_id=? AND organization_id=? AND kind='organizer' ORDER BY id`, eventID, organizationID)
	if err != nil {
		return summary, nil, err
	}
	defer rows.Close()
	notifications := []VotingNotification{}
	for rows.Next() {
		var n VotingNotification
		if err := rows.Scan(&n.ID, &n.Name, &n.Phone, &n.Status, &n.ProviderStatus, &n.Error); err != nil {
			return summary, nil, err
		}
		notifications = append(notifications, n)
	}
	return summary, notifications, rows.Err()
}

// Claiming takes a write lock before selecting, so separate API instances
// cannot send the same entry. Interrupted sends are never blindly resent.
func (db *appdbimpl) ClaimVotingNotification() (VotingNotification, error) {
	var n VotingNotification
	tx, err := db.c.Begin()
	if err != nil {
		return n, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE voting_notifications SET status='unknown',error='Invio interrotto: verifica il registro Twilio.',updated_at=CURRENT_TIMESTAMP
		WHERE status='sending' AND updated_at<datetime('now','-5 minutes')`); err != nil {
		return n, err
	}
	if err := tx.QueryRow(`SELECT id FROM voting_notifications WHERE status='pending' ORDER BY id LIMIT 1`).Scan(&n.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if commitErr := tx.Commit(); commitErr != nil {
				return n, commitErr
			}
		}
		return n, err
	}
	if _, err := tx.Exec(`UPDATE voting_notifications SET status='sending',updated_at=CURRENT_TIMESTAMP WHERE id=? AND status='pending'`, n.ID); err != nil {
		return n, err
	}
	var data string
	err = tx.QueryRow(`SELECT id,event_id,organization_id,kind,prize_id,name,phone,template_sid,variables_json FROM voting_notifications WHERE id=?`, n.ID).Scan(&n.ID, &n.EventID, &n.OrganizationID, &n.Kind, &n.PrizeID, &n.Name, &n.Phone, &n.TemplateSID, &data)
	if err != nil {
		return n, err
	}
	if err = json.Unmarshal([]byte(data), &n.Variables); err != nil {
		return n, err
	}
	return n, tx.Commit()
}

func (db *appdbimpl) FinishVotingNotification(id int, status, sid, providerStatus, message string) error {
	_, err := db.c.Exec(`UPDATE voting_notifications SET status=?,provider_sid=?,provider_status=?,error=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status='sending'`, status, sid, providerStatus, message, id)
	return err
}

func (db *appdbimpl) RetryVotingNotifications(eventID, organizationID int) error {
	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	settings, err := loadVotingSettings(tx, organizationID)
	if err != nil {
		return err
	}
	for _, recipient := range settings.Recipients {
		if !recipient.Enabled {
			continue
		}
		if _, err := tx.Exec(`UPDATE voting_notifications SET status='pending',template_sid=?,error='',updated_at=CURRENT_TIMESTAMP
			WHERE event_id=? AND organization_id=? AND phone=? AND kind='organizer' AND status IN ('failed','skipped')`, settings.TemplateSID, eventID, organizationID, recipient.Phone); err != nil {
			return err
		}
	}
	return tx.Commit()
}
