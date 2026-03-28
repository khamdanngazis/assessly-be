-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK(role IN ('creator', 'reviewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX users_email_idx ON users(email);
CREATE INDEX users_role_idx ON users(role);

-- Add comments
COMMENT ON TABLE users IS 'Stores authenticated users (creators and reviewers)';
COMMENT ON COLUMN users.email IS 'User email address (unique identifier for login)';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hashed password (cost factor 12)';
COMMENT ON COLUMN users.role IS 'User role: creator (creates tests) or reviewer (reviews submissions)';
