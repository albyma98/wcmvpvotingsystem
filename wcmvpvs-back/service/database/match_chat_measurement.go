package database

import (
	"context"
	"time"
)

func (s MatchChatStore) RecordOpen(ctx context.Context, fanID int, deviceKey string, now time.Time) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO event_chat_opens(event_id,device_key,fan_id,first_opened_at)
		VALUES(?,?,?,?) ON CONFLICT(event_id,device_key,fan_id) DO NOTHING`, s.EventID, deviceKey, fanID, now.UnixMilli())
	return err
}

func (s MatchChatStore) RecordBlockedAttempt(ctx context.Context, reason string, now time.Time) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO event_chat_blocked_attempts(event_id,minute,reason,attempts)
		VALUES(?,?,?,1) ON CONFLICT(event_id,minute,reason) DO UPDATE SET attempts=attempts+1`, s.EventID, now.Unix()/60, reason)
	return err
}
