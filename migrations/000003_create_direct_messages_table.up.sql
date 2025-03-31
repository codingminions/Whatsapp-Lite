CREATE TABLE IF NOT EXISTS direct_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    delivered BOOLEAN DEFAULT FALSE,
    read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for retrieving messages sent by a user
CREATE INDEX idx_direct_messages_sender_id ON direct_messages(sender_id);
-- Index for retrieving messages received by a user
CREATE INDEX idx_direct_messages_recipient_id ON direct_messages(recipient_id);
-- Index for sorting messages by time
CREATE INDEX idx_direct_messages_created_at ON direct_messages(created_at);

-- Special index for conversation view - efficiently retrieve conversations between two users
-- Using LEAST/GREATEST ensures consistent ordering regardless of who is sender/recipient
CREATE INDEX idx_direct_messages_conversation ON direct_messages(
    LEAST(sender_id, recipient_id),
    GREATEST(sender_id, recipient_id),
    created_at DESC
);