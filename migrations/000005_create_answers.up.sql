-- Create answers table
CREATE TABLE answers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions(id),
    text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(submission_id, question_id)
);

-- Create indexes
CREATE INDEX answers_submission_id_idx ON answers(submission_id);
CREATE INDEX answers_question_id_idx ON answers(question_id);

-- Add comments
COMMENT ON TABLE answers IS 'Stores individual essay answers within a submission';
COMMENT ON COLUMN answers.text IS 'Participant essay answer text';
