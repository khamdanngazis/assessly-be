-- Fix Railway Database Schema - Add Missing Columns
-- Run this if tables already exist but missing some columns

-- Fix tests table
DO $$ 
BEGIN
    -- Add allow_retakes if not exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'tests' AND column_name = 'allow_retakes'
    ) THEN
        ALTER TABLE tests ADD COLUMN allow_retakes BOOLEAN NOT NULL DEFAULT false;
        COMMENT ON COLUMN tests.allow_retakes IS 'Whether participants can submit multiple times (default: false)';
    END IF;

    -- Add description if not exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'tests' AND column_name = 'description'
    ) THEN
        ALTER TABLE tests ADD COLUMN description TEXT;
    END IF;

    -- Add is_published if not exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'tests' AND column_name = 'is_published'
    ) THEN
        ALTER TABLE tests ADD COLUMN is_published BOOLEAN NOT NULL DEFAULT false;
        COMMENT ON COLUMN tests.is_published IS 'Whether test is active and accessible to participants';
    END IF;
END $$;

-- Fix questions table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'questions') THEN
        -- Add expected_answer if not exists
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'questions' AND column_name = 'expected_answer'
        ) THEN
            ALTER TABLE questions ADD COLUMN expected_answer TEXT;
        END IF;

        -- Add order_num if not exists
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'questions' AND column_name = 'order_num'
        ) THEN
            ALTER TABLE questions ADD COLUMN order_num INTEGER NOT NULL DEFAULT 1;
        END IF;
    END IF;
END $$;

-- Fix submissions table  
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'submissions') THEN
        -- Add total_score if not exists
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'submissions' AND column_name = 'total_score'
        ) THEN
            ALTER TABLE submissions ADD COLUMN total_score DECIMAL(5,2);
        END IF;

        -- Add participant_name if not exists
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'submissions' AND column_name = 'participant_name'
        ) THEN
            ALTER TABLE submissions ADD COLUMN participant_name VARCHAR(255);
        END IF;

        -- Add participant_email if not exists
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'submissions' AND column_name = 'participant_email'
        ) THEN
            ALTER TABLE submissions ADD COLUMN participant_email VARCHAR(255);
        END IF;
    END IF;
END $$;

-- Fix answers table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'answers') THEN
        -- Add text if not exists
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'answers' AND column_name = 'text'
        ) THEN
            ALTER TABLE answers ADD COLUMN text TEXT NOT NULL DEFAULT '';
        END IF;
    END IF;
END $$;

-- Fix reviews table
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'reviews') THEN
        -- Add ai_score if not exists
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'reviews' AND column_name = 'ai_score'
        ) THEN
            ALTER TABLE reviews ADD COLUMN ai_score DECIMAL(5,2);
        END IF;

        -- Add ai_feedback if not exists
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'reviews' AND column_name = 'ai_feedback'
        ) THEN
            ALTER TABLE reviews ADD COLUMN ai_feedback TEXT;
        END IF;

        -- Add manual_score if not exists
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'reviews' AND column_name = 'manual_score'
        ) THEN
            ALTER TABLE reviews ADD COLUMN manual_score DECIMAL(5,2);
            COMMENT ON COLUMN reviews.manual_score IS 'Manual review score (0-100, overrides AI score)';
        END IF;

        -- Add manual_feedback if not exists
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'reviews' AND column_name = 'manual_feedback'
        ) THEN
            ALTER TABLE reviews ADD COLUMN manual_feedback TEXT;
        END IF;

        -- Add reviewer_id if not exists
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'reviews' AND column_name = 'reviewer_id'
        ) THEN
            ALTER TABLE reviews ADD COLUMN reviewer_id UUID REFERENCES users(id);
        END IF;
    END IF;
END $$;

-- Create missing indexes
DO $$
BEGIN
    -- tests indexes
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'tests_published_idx') THEN
        CREATE INDEX tests_published_idx ON tests(is_published) WHERE is_published = true;
    END IF;

    -- questions indexes
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'questions') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'questions_test_order_idx') THEN
            CREATE UNIQUE INDEX questions_test_order_idx ON questions(test_id, order_num);
        END IF;
    END IF;

    -- submissions indexes
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'submissions') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'submissions_participant_email_idx') THEN
            CREATE INDEX submissions_participant_email_idx ON submissions(participant_email);
        END IF;
    END IF;

    -- reviews indexes
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'reviews') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'reviews_reviewer_idx') THEN
            CREATE INDEX reviews_reviewer_idx ON reviews(reviewer_id);
        END IF;
    END IF;
END $$;

-- Summary
SELECT 
    'Schema fixes completed!' as status,
    (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public') as total_tables,
    (SELECT COUNT(*) FROM pg_indexes WHERE schemaname = 'public') as total_indexes;
