-- Create submissions table
CREATE TABLE submissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    test_id UUID NOT NULL REFERENCES tests(id),
    access_email VARCHAR(255) NOT NULL,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ai_total_score NUMERIC(5,2),
    manual_total_score NUMERIC(5,2)
);

-- Create indexes
CREATE INDEX submissions_test_id_idx ON submissions(test_id);
CREATE INDEX submissions_email_idx ON submissions(test_id, access_email);

-- Add comments
COMMENT ON TABLE submissions IS 'Stores participant test submissions (anonymous)';
COMMENT ON COLUMN submissions.access_email IS 'Participant email (not foreign key - participants not in users table)';
COMMENT ON COLUMN submissions.ai_total_score IS 'Sum of AI scores across all answers (denormalized for performance)';
COMMENT ON COLUMN submissions.manual_total_score IS 'Sum of manual scores across all answers (null if not reviewed)';
