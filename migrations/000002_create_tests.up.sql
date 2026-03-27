-- Create tests table
CREATE TABLE tests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    creator_id UUID NOT NULL REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    allow_retakes BOOLEAN NOT NULL DEFAULT false,
    is_published BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX tests_creator_id_idx ON tests(creator_id);
CREATE INDEX tests_published_idx ON tests(is_published) WHERE is_published = true;

-- Add comments
COMMENT ON TABLE tests IS 'Stores test definitions created by users';
COMMENT ON COLUMN tests.creator_id IS 'Foreign key to users table - test owner';
COMMENT ON COLUMN tests.allow_retakes IS 'Whether participants can submit multiple times (default: false)';
COMMENT ON COLUMN tests.is_published IS 'Whether test is active and accessible to participants';
