-- Rollback della migrazione applicata da ensureMatchChatTables.
-- Arrestare il backend e conservare una copia del database prima di eseguirlo.
BEGIN IMMEDIATE;
DROP INDEX IF EXISTS engagement_report_chat_opens;
DROP TABLE IF EXISTS event_chat_reactions;
DROP TABLE IF EXISTS event_chat_reports;
DROP TABLE IF EXISTS event_chat_mutes;
DROP TABLE IF EXISTS event_chat_messages;
DROP TABLE IF EXISTS event_chat_settings;
COMMIT;
