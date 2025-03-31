DROP INDEX IF EXISTS idx_message_delivery_status_message_id;
DROP INDEX IF EXISTS idx_message_delivery_status_user_id;
DROP INDEX IF EXISTS idx_message_delivery_status_unread;
DROP TABLE IF EXISTS message_delivery_status;

DROP INDEX IF EXISTS idx_group_messages_group_id_created_at;
DROP INDEX IF EXISTS idx_group_messages_sender_id;
DROP TABLE IF EXISTS group_messages;