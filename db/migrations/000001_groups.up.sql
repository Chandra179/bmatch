CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- OAuth provider (e.g., 'google', 'github')
    provider VARCHAR(255) NOT NULL,

    -- oidc subject id
    subject_id VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    
    full_name VARCHAR(100),
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    skill_level VARCHAR(50) NOT NULL DEFAULT 'BEGINNER', -- BEGINNER, INTERMEDIATE, ADVANCED
    availability JSONB NOT NULL DEFAULT '[]'::jsonb, -- ["WEEKENDS", "WEEKDAYS", "EVENINGS"]
    intent VARCHAR(50) NOT NULL DEFAULT 'CASUAL', -- CASUAL, SERIOUS
    stats JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
    
);

CREATE TABLE IF NOT EXISTS groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    proposal TEXT NOT NULL, -- The "why" - goals and clear purpose
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    capacity INTEGER NOT NULL DEFAULT 5,
    current_count INTEGER NOT NULL DEFAULT 1, -- Owner is auto-member
    join_type VARCHAR(50) NOT NULL DEFAULT 'OPEN', -- OPEN, APPLICATION
    status VARCHAR(50) NOT NULL DEFAULT 'OPEN', -- OPEN, CLOSED, COMPLETED
    applications JSONB NOT NULL DEFAULT '[]'::jsonb, -- [{user_id, pitch, status, applied_at, decided_at}]
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_capacity CHECK (capacity > 0 AND capacity <= 10),
    CONSTRAINT chk_count CHECK (current_count >= 0 AND current_count <= capacity)
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'MEMBER', -- LEADER, MEMBER
    joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (group_id, user_id)
);

-- Indexes for performance
CREATE INDEX idx_users_tags_gin ON users USING GIN (tags);
CREATE INDEX idx_users_skill_level ON users(skill_level);

CREATE INDEX idx_groups_tags_gin ON groups USING GIN (tags);
CREATE INDEX idx_groups_owner_id ON groups(owner_id);
CREATE INDEX idx_groups_status ON groups(status);
CREATE INDEX idx_groups_join_type ON groups(join_type);
CREATE INDEX idx_groups_status_join_type ON groups(status, join_type);

CREATE INDEX idx_group_members_user_id ON group_members(user_id);
CREATE INDEX idx_group_members_group_id ON group_members(group_id);