CREATE TABLE IF NOT EXISTS group_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Efficient index for retrieving messages in a group, ordered by time
CREATE INDEX idx_group_messages_group_id_created_at ON group_messages(group_id, created_at DESC);
-- Index for finding messages sent by a specific user
CREATE INDEX idx_group_messages_sender_id ON group_messages(sender_id);

-- Table for tracking message delivery status for each recipient in a group
CREATE TABLE IF NOT EXISTS message_delivery_status (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES group_messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    delivered BOOLEAN DEFAULT FALSE,
    read BOOLEAN DEFAULT FALSE,
    delivered_at TIMESTAMP WITH TIME ZONE,
    read_at TIMESTAMP WITH TIME ZONE,
    -- Each message can have only one delivery status per user
    UNIQUE(message_id, user_id)
);

-- Index for updating delivery status of a message
CREATE INDEX idx_message_delivery_status_message_id ON message_delivery_status(message_id);
-- Index for retrieving read/unread messages for a user
CREATE INDEX idx_message_delivery_status_user_id ON message_delivery_status(user_id);
-- Index for unread message counts
CREATE INDEX idx_message_delivery_status_unread ON message_delivery_status(user_id, read) WHERE read = FALSE;