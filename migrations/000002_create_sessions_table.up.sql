CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token VARCHAR(255) NOT NULL,
    user_agent VARCHAR(255) NOT NULL,
    client_ip VARCHAR(50) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_active_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for retrieving all sessions for a user
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
-- Index for looking up sessions by refresh token
CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token);