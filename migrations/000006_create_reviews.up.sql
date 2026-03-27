-- Create reviews table
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    answer_id UUID UNIQUE NOT NULL REFERENCES answers(id) ON DELETE CASCADE,
    reviewer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ai_score NUMERIC(5,2) CHECK(ai_score >= 0 AND ai_score <= 100),
    ai_feedback TEXT,
    ai_scored_at TIMESTAMPTZ,
    manual_score NUMERIC(5,2) CHECK(manual_score >= 0 AND manual_score <= 100),
    manual_feedback TEXT,
    manual_scored_at TIMESTAMPTZ
);

-- Create indexes
CREATE UNIQUE INDEX reviews_answer_id_idx ON reviews(answer_id);
CREATE INDEX reviews_reviewer_id_idx ON reviews(reviewer_id);

-- Add comments
COMMENT ON TABLE reviews IS 'Stores AI and manual scoring for individual answers';
COMMENT ON COLUMN reviews.answer_id IS 'One review per answer (unique constraint)';
COMMENT ON COLUMN reviews.reviewer_id IS 'NULL when only AI scoring exists (no manual review yet)';
COMMENT ON COLUMN reviews.ai_score IS 'AI-generated score (0-100)';
COMMENT ON COLUMN reviews.manual_score IS 'Manual override score (0-100)';
