package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func ensureMatchChatTables(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS event_chat_settings (
			event_id INTEGER PRIMARY KEY REFERENCES events(id) ON DELETE CASCADE,
			enabled INTEGER NOT NULL DEFAULT 0 CHECK(enabled IN (0,1)),
			created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL,
			updated_by INTEGER REFERENCES admins(id) ON DELETE SET NULL
		);
		CREATE TABLE IF NOT EXISTS event_chat_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			author_user_id INTEGER REFERENCES fan_profiles(id) ON DELETE SET NULL,
			admin_id INTEGER REFERENCES admins(id) ON DELETE SET NULL,
			request_id TEXT NOT NULL,
			message_text TEXT NOT NULL,
			message_type TEXT NOT NULL CHECK(message_type IN ('fan','official','ai')),
			created_at INTEGER NOT NULL,
			moderation_status TEXT NOT NULL DEFAULT 'visible' CHECK(moderation_status IN ('visible','removed')),
			deleted_at INTEGER,
			UNIQUE(event_id,id), UNIQUE(event_id,request_id)
		);
		CREATE INDEX IF NOT EXISTS idx_event_chat_visible ON event_chat_messages(event_id,id) WHERE moderation_status='visible';
		CREATE INDEX IF NOT EXISTS idx_event_chat_author ON event_chat_messages(event_id,author_user_id,created_at);
		CREATE TABLE IF NOT EXISTS event_chat_reactions (
			event_id INTEGER NOT NULL, message_id INTEGER NOT NULL,
			fan_id INTEGER NOT NULL REFERENCES fan_profiles(id) ON DELETE CASCADE,
			emoji TEXT NOT NULL CHECK(emoji IN ('❤️','🔥','😂','👏','💪')),
			PRIMARY KEY(event_id,message_id,emoji,fan_id),
			FOREIGN KEY(event_id,message_id) REFERENCES event_chat_messages(event_id,id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS event_chat_reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id INTEGER NOT NULL, message_id INTEGER NOT NULL,
			reporter_user_id INTEGER REFERENCES fan_profiles(id) ON DELETE SET NULL,
			reason TEXT NOT NULL, created_at INTEGER NOT NULL,
			review_status TEXT NOT NULL DEFAULT 'pending' CHECK(review_status IN ('pending','reviewed')),
			UNIQUE(event_id,message_id,reporter_user_id),
			FOREIGN KEY(event_id,message_id) REFERENCES event_chat_messages(event_id,id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_event_chat_reports ON event_chat_reports(event_id,id);
		CREATE TABLE IF NOT EXISTS event_chat_mutes (
			event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			fan_id INTEGER NOT NULL REFERENCES fan_profiles(id) ON DELETE CASCADE,
			muted_until INTEGER NOT NULL,
			updated_by INTEGER REFERENCES admins(id) ON DELETE SET NULL,
			PRIMARY KEY(event_id,fan_id)
		);
		CREATE INDEX IF NOT EXISTS idx_event_chat_mutes ON event_chat_mutes(event_id,muted_until);
	`)
	if err != nil {
		return fmt.Errorf("creating match chat tables: %w", err)
	}
	return tx.Commit()
}

var (
	ErrChatDisabled        = errors.New("chat_disabled")
	ErrChatMuted           = errors.New("chat_muted")
	ErrChatRate            = errors.New("chat_rate_limited")
	ErrChatDuplicate       = errors.New("chat_duplicate")
	ErrChatRequestConflict = errors.New("chat_request_conflict")
)

const ChatPageSize = 30

type MatchChatStore struct {
	DB             *sql.DB
	EventID        int
	OrganizationID int
}

type ChatReaction struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
	Mine  bool   `json:"mine,omitempty"`
}

type ChatMessage struct {
	ID        int64          `json:"id"`
	Nickname  string         `json:"nickname"`
	Text      string         `json:"text"`
	Type      string         `json:"type"`
	CreatedAt int64          `json:"created_at"`
	Status    string         `json:"status"`
	Mine      bool           `json:"mine,omitempty"`
	AuthorID  int            `json:"author_id,omitempty"`
	Reactions []ChatReaction `json:"reactions,omitempty"`
}

type ChatHistory struct {
	Messages   []ChatMessage `json:"messages"`
	NextBefore int64         `json:"next_before"`
	CanWrite   bool          `json:"can_write"`
}

type ChatReport struct {
	ID            int64  `json:"id"`
	MessageID     int64  `json:"message_id"`
	MessageStatus string `json:"message_status"`
	Reason        string `json:"reason"`
	Status        string `json:"status"`
	CreatedAt     int64  `json:"created_at"`
	Nickname      string `json:"nickname"`
	Text          string `json:"text"`
}

type ChatMute struct {
	FanID    int    `json:"fan_id"`
	Nickname string `json:"nickname"`
	Until    int64  `json:"until"`
}

type ChatStats struct {
	Messages        int `json:"messages"`
	Authors         int `json:"authors"`
	Reactions       int `json:"reactions"`
	Reports         int `json:"reports"`
	Opens           int `json:"opens"`
	OpenersWhoWrote int `json:"openers_who_wrote"`
}

type chatQueryer interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func (s MatchChatStore) enabled(ctx context.Context, q chatQueryer) (bool, error) {
	var enabled bool
	err := q.QueryRowContext(ctx, `SELECT COALESCE(c.enabled,0)
		FROM events e LEFT JOIN event_chat_settings c ON c.event_id=e.id
		WHERE e.id=? AND e.organization_id=?`, s.EventID, s.OrganizationID).Scan(&enabled)
	return enabled, err
}

func (s MatchChatStore) Enabled(ctx context.Context) (bool, error) {
	return s.enabled(ctx, s.DB)
}

// Resolve the existing OTP session without touching wallets or exposing profile data.
func (s MatchChatStore) VerifiedFan(ctx context.Context, token string) (int, string, error) {
	var id int
	var nickname string
	err := s.DB.QueryRowContext(ctx, `SELECT p.id,p.nickname FROM fan_sessions f JOIN fan_profiles p ON p.id=f.fan_id
		WHERE f.token=? AND p.organization_id=? AND p.registration_pending=0 AND p.accepted_terms=1
		AND TRIM(p.nickname)<>'' AND p.phone=p.phone_e164 AND TRIM(p.phone_verified_at)<>''`, token, s.OrganizationID).Scan(&id, &nickname)
	return id, nickname, err
}

// The first statement takes SQLite's writer lock before reading state. This
// avoids a deferred read transaction racing with a toggle or another writer.
func (s MatchChatStore) Write(ctx context.Context, requireEnabled bool, fn func(*sql.Tx, bool) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE event_chat_settings SET enabled=enabled WHERE event_id=?`, s.EventID); err != nil {
		return err
	}
	enabled, err := s.enabled(ctx, tx)
	if err != nil {
		return err
	}
	if requireEnabled && !enabled {
		return ErrChatDisabled
	}
	if err := fn(tx, enabled); err != nil {
		return err
	}
	return tx.Commit()
}

func (s MatchChatStore) SetEnabled(ctx context.Context, tx *sql.Tx, enabled bool, admin int, now int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO event_chat_settings(event_id,enabled,created_at,updated_at,updated_by)
		VALUES(?,?,?,?,?) ON CONFLICT(event_id) DO UPDATE SET enabled=excluded.enabled,
		updated_at=excluded.updated_at,updated_by=excluded.updated_by`, s.EventID, enabled, now, now, admin)
	return err
}

const chatMessageColumns = `m.id, COALESCE(NULLIF(p.nickname,''),'Tifoso'), m.message_text, m.message_type,
	m.created_at, m.moderation_status, COALESCE(m.author_user_id,0), COALESCE(m.admin_id,0)`

func scanChatMessage(row rowScanner, fanID int, admin bool) (ChatMessage, int, error) {
	var m ChatMessage
	var author, adminID int
	err := row.Scan(&m.ID, &m.Nickname, &m.Text, &m.Type, &m.CreatedAt, &m.Status, &author, &adminID)
	if m.Type == "official" {
		m.Nickname = "Società"
	}
	m.Mine = fanID > 0 && fanID == author
	if admin {
		m.AuthorID = author
	}
	return m, adminID, err
}

func (s MatchChatStore) History(ctx context.Context, before int64, fanID int, admin bool) (ChatHistory, error) {
	h := ChatHistory{Messages: []ChatMessage{}, CanWrite: fanID > 0}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return h, err
	}
	defer tx.Rollback()
	enabled, err := s.enabled(ctx, tx)
	if err != nil {
		return h, err
	}
	if !admin && !enabled {
		return h, ErrChatDisabled
	}
	query := `SELECT ` + chatMessageColumns + ` FROM event_chat_messages m
		LEFT JOIN fan_profiles p ON p.id=m.author_user_id
		WHERE m.event_id=?`
	args := []interface{}{s.EventID}
	if before > 0 {
		query += " AND m.id<?"
		args = append(args, before)
	}
	if !admin {
		query += " AND m.moderation_status='visible'"
	}
	rows, err := tx.QueryContext(ctx, query+" ORDER BY m.id DESC LIMIT ?", append(args, ChatPageSize+1)...)
	if err != nil {
		return h, err
	}
	defer rows.Close()
	for rows.Next() {
		m, _, err := scanChatMessage(rows, fanID, admin)
		if err != nil {
			return h, err
		}
		h.Messages = append(h.Messages, m)
	}
	if err := rows.Err(); err != nil {
		return h, err
	}
	if len(h.Messages) > ChatPageSize {
		h.Messages = h.Messages[:ChatPageSize]
		h.NextBefore = h.Messages[len(h.Messages)-1].ID
	}
	if len(h.Messages) > 0 {
		if err := s.loadReactions(ctx, tx, h.Messages, fanID); err != nil {
			return h, err
		}
	}
	for i, j := 0, len(h.Messages)-1; i < j; i, j = i+1, j-1 {
		h.Messages[i], h.Messages[j] = h.Messages[j], h.Messages[i]
	}
	return h, tx.Commit()
}

func (s MatchChatStore) loadReactions(ctx context.Context, tx *sql.Tx, messages []ChatMessage, fanID int) error {
	args := []interface{}{fanID, s.EventID}
	byID := make(map[int64]*ChatMessage, len(messages))
	for i := range messages {
		byID[messages[i].ID] = &messages[i]
		args = append(args, messages[i].ID)
	}
	rows, err := tx.QueryContext(ctx, `SELECT message_id,emoji,COUNT(*),MAX(fan_id=?)
		FROM event_chat_reactions WHERE event_id=? AND message_id IN (?`+strings.Repeat(",?", len(messages)-1)+`)
		GROUP BY message_id,emoji`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var reaction ChatReaction
		if err := rows.Scan(&id, &reaction.Emoji, &reaction.Count, &reaction.Mine); err != nil {
			return err
		}
		m := byID[id]
		m.Reactions = append(m.Reactions, reaction)
	}
	return rows.Err()
}

type ChatSend struct {
	FanID           int
	AdminID         int
	RequestID       string
	Text            string
	Nickname        string
	Now             int64
	MinInterval     time.Duration
	PerMinute       int
	DuplicateWindow time.Duration
}

func (s MatchChatStore) Send(ctx context.Context, tx *sql.Tx, in ChatSend) (ChatMessage, bool, error) {
	previous, adminID, err := scanChatMessage(tx.QueryRowContext(ctx, `SELECT `+chatMessageColumns+`
		FROM event_chat_messages m LEFT JOIN fan_profiles p ON p.id=m.author_user_id
		WHERE m.event_id=? AND m.request_id=?`, s.EventID, in.RequestID), in.FanID, false)
	if err == nil {
		if previous.Text != in.Text || (in.FanID > 0 && !previous.Mine) || (in.AdminID > 0 && in.AdminID != adminID) {
			return ChatMessage{}, false, ErrChatRequestConflict
		}
		return previous, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ChatMessage{}, false, err
	}

	kind := "official"
	nickname := "Società"
	if in.FanID > 0 {
		kind, nickname = "fan", in.Nickname
		if err := s.CheckMute(ctx, tx, in.FanID, in.Now); err != nil {
			return ChatMessage{}, false, err
		}
		var last int64
		var count int
		err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(created_at),0),COUNT(*) FROM event_chat_messages
			WHERE event_id=? AND author_user_id=? AND created_at>?`, s.EventID, in.FanID, in.Now-time.Minute.Milliseconds()).Scan(&last, &count)
		if err != nil {
			return ChatMessage{}, false, err
		}
		if count >= in.PerMinute || (last > 0 && in.Now-last < in.MinInterval.Milliseconds()) {
			return ChatMessage{}, false, ErrChatRate
		}
		// Compare in Go so case folding works for non-ASCII nicknames and text.
		rows, err := tx.QueryContext(ctx, `SELECT message_text FROM event_chat_messages
			WHERE event_id=? AND author_user_id=? AND created_at>?`, s.EventID, in.FanID, in.Now-in.DuplicateWindow.Milliseconds())
		if err != nil {
			return ChatMessage{}, false, err
		}
		defer rows.Close()
		for rows.Next() {
			var text string
			if err := rows.Scan(&text); err != nil {
				return ChatMessage{}, false, err
			}
			if strings.EqualFold(strings.Join(strings.Fields(text), " "), strings.Join(strings.Fields(in.Text), " ")) {
				return ChatMessage{}, false, ErrChatDuplicate
			}
		}
		if err := rows.Err(); err != nil {
			return ChatMessage{}, false, err
		}
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO event_chat_messages
		(event_id,author_user_id,admin_id,request_id,message_text,message_type,created_at)
		VALUES(?,NULLIF(?,0),NULLIF(?,0),?,?,?,?)`, s.EventID, in.FanID, in.AdminID, in.RequestID, in.Text, kind, in.Now)
	if err != nil {
		return ChatMessage{}, false, err
	}
	id, err := result.LastInsertId()
	return ChatMessage{ID: id, Nickname: nickname, Text: in.Text, Type: kind, CreatedAt: in.Now, Status: "visible", Mine: in.FanID > 0}, true, err
}

func (s MatchChatStore) CheckMute(ctx context.Context, tx *sql.Tx, fanID int, now int64) error {
	var muted bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM event_chat_mutes
		WHERE event_id=? AND fan_id=? AND muted_until>?)`, s.EventID, fanID, now).Scan(&muted)
	if err == nil && muted {
		return ErrChatMuted
	}
	return err
}

func (s MatchChatStore) React(ctx context.Context, tx *sql.Tx, messageID int64, fanID int, emoji string, active bool) (ChatReaction, bool, error) {
	r := ChatReaction{Emoji: emoji, Mine: active}
	if err := s.visibleMessage(ctx, tx, messageID); err != nil {
		return r, false, err
	}
	query := `DELETE FROM event_chat_reactions WHERE event_id=? AND message_id=? AND fan_id=? AND emoji=?`
	if active {
		query = `INSERT INTO event_chat_reactions(event_id,message_id,fan_id,emoji) VALUES(?,?,?,?) ON CONFLICT DO NOTHING`
	}
	result, err := tx.ExecContext(ctx, query, s.EventID, messageID, fanID, emoji)
	if err != nil {
		return r, false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return r, false, err
	}
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_chat_reactions
		WHERE event_id=? AND message_id=? AND emoji=?`, s.EventID, messageID, emoji).Scan(&r.Count)
	return r, n > 0, err
}

func (s MatchChatStore) visibleMessage(ctx context.Context, tx *sql.Tx, id int64) error {
	var found int
	return tx.QueryRowContext(ctx, `SELECT 1 FROM event_chat_messages WHERE event_id=? AND id=? AND moderation_status='visible'`, s.EventID, id).Scan(&found)
}

func (s MatchChatStore) Report(ctx context.Context, tx *sql.Tx, messageID int64, fanID int, reason string, now int64) (bool, error) {
	if err := s.visibleMessage(ctx, tx, messageID); err != nil {
		return false, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO event_chat_reports(event_id,message_id,reporter_user_id,reason,created_at)
		VALUES(?,?,?,?,?) ON CONFLICT DO NOTHING`, s.EventID, messageID, fanID, reason, now)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

func (s MatchChatStore) Remove(ctx context.Context, tx *sql.Tx, id int64, now int64) (bool, error) {
	result, err := tx.ExecContext(ctx, `UPDATE event_chat_messages SET moderation_status='removed',deleted_at=?
		WHERE event_id=? AND id=? AND moderation_status='visible'`, now, s.EventID, id)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 0 {
		var exists int
		return false, tx.QueryRowContext(ctx, `SELECT 1 FROM event_chat_messages WHERE event_id=? AND id=?`, s.EventID, id).Scan(&exists)
	}
	return true, nil
}

func (s MatchChatStore) Review(ctx context.Context, tx *sql.Tx, id int64) error {
	result, err := tx.ExecContext(ctx, `UPDATE event_chat_reports SET review_status='reviewed' WHERE event_id=? AND id=?`, s.EventID, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return err
}

func (s MatchChatStore) SetMute(ctx context.Context, tx *sql.Tx, fanID, adminID int, until int64) error {
	if until == 0 {
		_, err := tx.ExecContext(ctx, `DELETE FROM event_chat_mutes WHERE event_id=? AND fan_id=?`, s.EventID, fanID)
		return err
	}
	var exists int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM event_chat_messages m JOIN fan_profiles p ON p.id=m.author_user_id
		WHERE m.event_id=? AND m.author_user_id=? AND p.organization_id=? LIMIT 1`, s.EventID, fanID, s.OrganizationID).Scan(&exists)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO event_chat_mutes(event_id,fan_id,muted_until,updated_by) VALUES(?,?,?,?)
		ON CONFLICT(event_id,fan_id) DO UPDATE SET muted_until=excluded.muted_until,updated_by=excluded.updated_by`, s.EventID, fanID, until, adminID)
	return err
}

func (s MatchChatStore) Reports(ctx context.Context, before int64) ([]ChatReport, int64, error) {
	query := `SELECT r.id,r.message_id,m.moderation_status,r.reason,r.review_status,r.created_at,
		CASE WHEN m.message_type='official' THEN 'Società' ELSE COALESCE(p.nickname,'Tifoso') END,m.message_text
		FROM event_chat_reports r JOIN event_chat_messages m ON m.event_id=r.event_id AND m.id=r.message_id
		LEFT JOIN fan_profiles p ON p.id=m.author_user_id WHERE r.event_id=?`
	args := []interface{}{s.EventID}
	if before > 0 {
		query += " AND r.id<?"
		args = append(args, before)
	}
	rows, err := s.DB.QueryContext(ctx, query+" ORDER BY r.id DESC LIMIT ?", append(args, ChatPageSize+1)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	reports := []ChatReport{}
	for rows.Next() {
		var item ChatReport
		if err := rows.Scan(&item.ID, &item.MessageID, &item.MessageStatus, &item.Reason, &item.Status, &item.CreatedAt, &item.Nickname, &item.Text); err != nil {
			return nil, 0, err
		}
		reports = append(reports, item)
	}
	var next int64
	if len(reports) > ChatPageSize {
		reports = reports[:ChatPageSize]
		next = reports[len(reports)-1].ID
	}
	return reports, next, rows.Err()
}

func (s MatchChatStore) Mutes(ctx context.Context, now int64, after int) ([]ChatMute, int, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT m.fan_id,p.nickname,m.muted_until FROM event_chat_mutes m
		JOIN fan_profiles p ON p.id=m.fan_id WHERE m.event_id=? AND m.muted_until>? AND m.fan_id>?
		ORDER BY m.fan_id LIMIT ?`, s.EventID, now, after, ChatPageSize+1)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	mutes := []ChatMute{}
	for rows.Next() {
		var item ChatMute
		if err := rows.Scan(&item.FanID, &item.Nickname, &item.Until); err != nil {
			return nil, 0, err
		}
		mutes = append(mutes, item)
	}
	var next int
	if len(mutes) > ChatPageSize {
		mutes = mutes[:ChatPageSize]
		next = mutes[len(mutes)-1].FanID
	}
	return mutes, next, rows.Err()
}

func (s MatchChatStore) Stats(ctx context.Context) (ChatStats, error) {
	var stats ChatStats
	err := s.DB.QueryRowContext(ctx, `WITH authors AS (
		SELECT DISTINCT author_user_id FROM event_chat_messages WHERE event_id=?1 AND author_user_id IS NOT NULL
	), open_signals AS (
		SELECT device_key,fan_id FROM engagement_report_signals WHERE event_id=?1 AND name='match_chat_opened'
	), verified_devices AS (
		SELECT device_key,MAX(fan_id) AS fan_id FROM open_signals WHERE fan_id>0 GROUP BY device_key
	), opens AS (
		SELECT DISTINCT CASE WHEN s.fan_id>0 THEN 'fan:'||s.fan_id
			ELSE COALESCE('fan:'||v.fan_id,'device:'||s.device_key) END AS identity
		FROM open_signals s LEFT JOIN verified_devices v ON v.device_key=s.device_key
	)
	SELECT (SELECT COUNT(*) FROM event_chat_messages WHERE event_id=?1),
		(SELECT COUNT(*) FROM authors), (SELECT COUNT(*) FROM event_chat_reactions WHERE event_id=?1),
		(SELECT COUNT(*) FROM event_chat_reports WHERE event_id=?1), (SELECT COUNT(*) FROM opens),
		(SELECT COUNT(*) FROM opens JOIN authors ON identity='fan:'||author_user_id)`, s.EventID).
		Scan(&stats.Messages, &stats.Authors, &stats.Reactions, &stats.Reports, &stats.Opens, &stats.OpenersWhoWrote)
	return stats, err
}
