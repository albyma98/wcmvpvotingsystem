package api

import (
	"context"
	"database/sql"
)

func loadMatchdayChatMeasurement(ctx context.Context, q matchdayQueryer, orgID int, data map[int]*matchdayFacts) error {
	ids, args := matchdayIDs(data)
	args = append(args, orgID)
	if err := matchdayRead(ctx, q, `SELECT e.id, EXISTS(SELECT 1 FROM event_chat_setting_changes c WHERE c.event_id=e.id AND c.enabled=1),
		EXISTS(SELECT 1 FROM event_chat_settings s JOIN event_chat_setting_changes c ON c.event_id=s.event_id AND c.changed_at=s.created_at WHERE s.event_id=e.id),
		(SELECT COUNT(*) FROM event_chat_replies r WHERE r.event_id=e.id),
		COALESCE((SELECT SUM(b.attempts) FROM event_chat_blocked_attempts b WHERE b.event_id=e.id),0)
		FROM events e WHERE e.id IN (`+ids+`) AND e.organization_id=?`, args, func(rows *sql.Rows) error {
		var eventID, replies, blocked int
		var everEnabled, complete bool
		if err := rows.Scan(&eventID, &everEnabled, &complete, &replies, &blocked); err != nil {
			return err
		}
		chat := &data[eventID].Operations.chat
		chat.measured, chat.everEnabled, chat.replies, chat.blocked = everEnabled, everEnabled, replies, blocked
		chat.complete = complete
		return nil
	}); err != nil {
		return err
	}
	return matchdayRead(ctx, q, `SELECT o.event_id,COALESCE(p.id,0),o.device_key FROM event_chat_opens o
		JOIN events e ON e.id=o.event_id LEFT JOIN fan_profiles p ON p.id=o.fan_id AND p.organization_id=e.organization_id
		WHERE o.event_id IN (`+ids+`) AND e.organization_id=?`, args, func(rows *sql.Rows) error {
		var eventID int
		var identity matchdayIdentity
		if err := rows.Scan(&eventID, &identity.FanID, &identity.DeviceKey); err != nil {
			return err
		}
		chat := &data[eventID].Operations.chat
		chat.measured = true
		if identity.FanID == 0 && identity.DeviceKey == "" {
			chat.openWithoutIdentity = true
		} else {
			chat.openIdentities = append(chat.openIdentities, identity)
		}
		return nil
	})
}
