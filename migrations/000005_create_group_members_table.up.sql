-- Create enum type for member roles
CREATE TYPE member_role AS ENUM ('admin', 'member');

CREATE TABLE IF NOT EXISTS group_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role member_role NOT NULL DEFAULT 'member',
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    -- Ensure a user can only be a member of a group once
    UNIQUE(group_id, user_id)
);

-- Index for finding all members of a specific group
CREATE INDEX idx_group_members_group_id ON group_members(group_id);
-- Index for finding all groups a user is a member of
CREATE INDEX idx_group_members_user_id ON group_members(user_id);
-- Index for finding admins of a group
CREATE INDEX idx_group_members_role ON group_members(group_id, role);