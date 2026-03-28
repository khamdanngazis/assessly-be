-- Create questions table
CREATE TABLE questions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    test_id UUID NOT NULL REFERENCES tests(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    expected_answer TEXT NOT NULL,
    order_num INTEGER NOT NULL CHECK(order_num > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(test_id, order_num)
);

-- Create indexes
CREATE INDEX questions_test_id_order_idx ON questions(test_id, order_num);

-- Add comments
COMMENT ON TABLE questions IS 'Stores essay questions belonging to tests';
COMMENT ON COLUMN questions.expected_answer IS 'Expected answer or rubric for AI scoring';
COMMENT ON COLUMN questions.order_num IS 'Display order (1-based, unique within test)';
