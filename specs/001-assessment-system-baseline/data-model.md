# Data Model: Flexible Assessment System Baseline

**Feature**: 001-assessment-system-baseline  
**Phase**: 1 (Design & Contracts)  
**Date**: 2026-03-26

## Overview

This document defines the database schema, entity relationships, and data constraints for the baseline assessment system. The model supports:
- User management (creators and reviewers only)
- Test creation with multiple essay questions
- Anonymous participant submissions without authentication
- Dual review system (AI scoring + manual review)
- Per-answer scoring granularity
- Email-based access tokens for participant test access

## Entity-Relationship Diagram

```
┌──────────────┐
│    User      │
│--------------│
│ id (PK)      │───┐
│ email        │   │
│ password_hash│   │
│ role         │   │
│ created_at   │   │
└──────────────┘   │
                   │ 1:N (creator_id)
                   │
                   ▼
         ┌─────────────────┐
         │      Test       │
         │-----------------│
         │ id (PK)         │───┐
         │ creator_id (FK) │   │
         │ title           │   │ 1:N
         │ description     │   │
         │ allow_retakes   │   │
         │ is_published    │   ▼
         │ created_at      │   ┌──────────────┐
         │ updated_at      │   │   Question   │
         └─────────────────┘   │--------------│
                   │           │ id (PK)      │
                   │           │ test_id (FK) │
                   │ 1:N       │ text         │
                   │           │ expected     │
                   │           │ order_num    │
                   ▼           │ created_at   │
         ┌─────────────────┐  └──────────────┘
         │   Submission    │           │
         │-----------------│           │
         │ id (PK)         │           │ 1:N
         │ test_id (FK)    │           │
         │ access_email    │           ▼
         │ submitted_at    │   ┌──────────────┐
         │ ai_total_score  │   │    Answer    │
         │ manual_total    │   │--------------│
         └─────────────────┘   │ id (PK)      │
                   │           │ submission_id│───┐
                   │ 1:N       │ question_id  │   │
                   │           │ text         │   │
                   │           │ created_at   │   │
                   │           └──────────────┘   │
                   │                              │
                   ▼                              │ 1:1
         ┌─────────────────┐                     │
         │     Review      │◄────────────────────┘
         │-----------------│
         │ id (PK)         │
         │ answer_id (FK)  │
         │ reviewer_id (FK)│───┐
         │ ai_score        │   │
         │ ai_feedback     │   │
         │ manual_score    │   │ N:1
         │ manual_feedback │   │
         │ reviewed_at     │   │
         └─────────────────┘   │
                               │
                               ▼
                     ┌──────────────┐
                     │    User      │
                     │ (reviewer)   │
                     └──────────────┘
```

## Database Schema

### Table: users

**Purpose**: Stores authenticated users (creators and reviewers)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique user identifier |
| email | VARCHAR(255) | UNIQUE NOT NULL | User email address |
| password_hash | VARCHAR(255) | NOT NULL | Bcrypt hashed password |
| role | VARCHAR(20) | NOT NULL CHECK(role IN ('creator', 'reviewer')) | User role |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Account creation timestamp |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Last update timestamp |

**Indexes**:
- PRIMARY KEY: `id`
- UNIQUE INDEX: `users_email_idx` ON `email`
- INDEX: `users_role_idx` ON `role` (for reviewer queries)

**Notes**:
- Email uniqueness enforced at database level
- Password hash uses bcrypt (cost factor 12)
- No soft deletes in baseline (future: deleted_at column)

---

### Table: tests

**Purpose**: Stores test definitions created by users

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique test identifier |
| creator_id | UUID | FOREIGN KEY REFERENCES users(id) NOT NULL | Test creator |
| title | VARCHAR(255) | NOT NULL | Test title |
| description | TEXT | NULL | Test description/instructions |
| allow_retakes | BOOLEAN | NOT NULL DEFAULT false | Whether multiple submissions allowed |
| is_published | BOOLEAN | NOT NULL DEFAULT false | Publish status |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Test creation timestamp |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Last update timestamp |

**Indexes**:
- PRIMARY KEY: `id`
- INDEX: `tests_creator_id_idx` ON `creator_id` (for user's tests query)
- INDEX: `tests_published_idx` ON `is_published` WHERE `is_published = true` (for published tests list)

**Notes**:
- Unpublished tests are drafts (not visible to participants)
- `allow_retakes` flag from clarification #2 (default: false)
- Soft delete future enhancement (deleted_at column)

---

### Table: questions

**Purpose**: Stores essay questions belonging to tests

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique question identifier |
| test_id | UUID | FOREIGN KEY REFERENCES tests(id) ON DELETE CASCADE NOT NULL | Parent test |
| text | TEXT | NOT NULL | Question prompt |
| expected_answer | TEXT | NOT NULL | Expected answer/rubric for AI scoring |
| order_num | INTEGER | NOT NULL | Display order (1-based) |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Question creation timestamp |

**Indexes**:
- PRIMARY KEY: `id`
- INDEX: `questions_test_id_order_idx` ON `(test_id, order_num)` (for ordered retrieval)

**Constraints**:
- UNIQUE: `(test_id, order_num)` (no duplicate order within test)
- CHECK: `order_num > 0` (order must be positive)

**Notes**:
- ON DELETE CASCADE: Deleting test deletes its questions
- `expected_answer` stores rubric/guidelines for AI scorer (clarification #3)
- Order preserved for consistent presentation to participants

---

### Table: submissions

**Purpose**: Stores participant test submissions (anonymous)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique submission identifier |
| test_id | UUID | FOREIGN KEY REFERENCES tests(id) NOT NULL | Test taken |
| access_email | VARCHAR(255) | NOT NULL | Participant email (for access token) |
| submitted_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Submission timestamp |
| ai_total_score | NUMERIC(5,2) | NULL | Sum of AI scores across answers |
| manual_total_score | NUMERIC(5,2) | NULL | Sum of manual scores (if reviewed) |

**Indexes**:
- PRIMARY KEY: `id`
- INDEX: `submissions_test_id_idx` ON `test_id` (for test's submissions query)
- INDEX: `submissions_email_idx` ON `(test_id, access_email)` (for retake checks)

**Notes**:
- `access_email` is NOT a foreign key (participants not in users table)
- `ai_total_score` and `manual_total_score` are denormalized aggregates (for performance)
- Updated via trigger or application logic when reviews are created/updated
- Check retakes: Query existing submissions for (test_id, access_email)

---

### Table: answers

**Purpose**: Stores individual essay answers within a submission

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique answer identifier |
| submission_id | UUID | FOREIGN KEY REFERENCES submissions(id) ON DELETE CASCADE NOT NULL | Parent submission |
| question_id | UUID | FOREIGN KEY REFERENCES questions(id) NOT NULL | Question answered |
| text | TEXT | NOT NULL | Participant's essay text |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Answer creation timestamp |

**Indexes**:
- PRIMARY KEY: `id`
- INDEX: `answers_submission_id_idx` ON `submission_id` (for submission's answers)
- INDEX: `answers_question_id_idx` ON `question_id` (for question analysis)

**Constraints**:
- UNIQUE: `(submission_id, question_id)` (one answer per question per submission)

**Notes**:
- ON DELETE CASCADE: Deleting submission deletes its answers
- No FK to users (anonymous participant)
- Answers can be empty (TEXT allows empty string) but NOT NULL enforced

---

### Table: reviews

**Purpose**: Stores AI and manual scoring for individual answers

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique review identifier |
| answer_id | UUID | FOREIGN KEY REFERENCES answers(id) ON DELETE CASCADE UNIQUE NOT NULL | Answer being reviewed |
| reviewer_id | UUID | FOREIGN KEY REFERENCES users(id) NULL | Reviewer (NULL for AI-only) |
| ai_score | NUMERIC(5,2) | NULL CHECK(ai_score >= 0 AND ai_score <= 100) | AI-generated score (0-100) |
| ai_feedback | TEXT | NULL | AI-generated feedback |
| ai_scored_at | TIMESTAMPTZ | NULL | AI scoring timestamp |
| manual_score | NUMERIC(5,2) | NULL CHECK(manual_score >= 0 AND manual_score <= 100) | Manual override score (0-100) |
| manual_feedback | TEXT | NULL | Reviewer's feedback |
| manual_scored_at | TIMESTAMPTZ | NULL | Manual review timestamp |

**Indexes**:
- PRIMARY KEY: `id`
- UNIQUE INDEX: `reviews_answer_id_idx` ON `answer_id` (one review per answer)
- INDEX: `reviews_reviewer_id_idx` ON `reviewer_id` (for reviewer's work history)

**Notes**:
- **Per-answer scoring**: Each answer has exactly one review (clarification #5)
- `reviewer_id` NULL when only AI scoring exists (no manual review yet)
- Both AI and manual scores can coexist (compare AI vs human judgment)
- Scores constrained to 0-100 range at database level
- Timestamps differentiate AI vs manual review timing

---

## Relationships Summary

| Parent | Child | Relationship | Cascade |
|--------|-------|--------------|---------|
| users | tests | 1:N (creator_id) | RESTRICT (default) |
| tests | questions | 1:N (test_id) | CASCADE |
| tests | submissions | 1:N (test_id) | RESTRICT |
| submissions | answers | 1:N (submission_id) | CASCADE |
| questions | answers | 1:N (question_id) | RESTRICT |
| answers | reviews | 1:1 (answer_id) | CASCADE |
| users | reviews | 1:N (reviewer_id) | SET NULL |

**Cascade Behavior**:
- DELETE test → DELETE questions (test content bundle)
- DELETE submission → DELETE answers → DELETE reviews (submission bundle)
- DELETE user → RESTRICT if tests exist, SET NULL on reviews (preserve audit trail)

---

## Data Integrity Constraints

### Application-Level Constraints

1. **Retake Prevention** (FR-007):
   ```sql
   -- Before insert submission:
   SELECT COUNT(*) FROM submissions 
   WHERE test_id = ? AND access_email = ?;
   
   -- If count > 0 AND test.allow_retakes = false:
   --   REJECT submission
   ```

2. **Total Score Calculation** (FR-017, FR-027):
   ```sql
   -- After insert/update review for answer:
   UPDATE submissions
   SET ai_total_score = (
     SELECT COALESCE(SUM(r.ai_score), 0)
     FROM answers a
     JOIN reviews r ON r.answer_id = a.id
     WHERE a.submission_id = ?
   ),
   manual_total_score = (
     SELECT COALESCE(SUM(r.manual_score), 0)
     FROM answers a
     JOIN reviews r ON r.answer_id = a.id
     WHERE a.submission_id = ? AND r.manual_score IS NOT NULL
   )
   WHERE id = ?;
   ```

3. **Access Control** (Clarification #4):
   - Creators can only access tests WHERE `creator_id = current_user_id`
   - Reviewers can access all published tests (no WHERE clause on creator)
   - Enforce in usecase layer, not database

### Database-Level Constraints

- **Email Format**: Validated at application layer (regex), not database
- **Non-empty Text**: `text` columns use TEXT (allows empty), but application validates non-empty
- **Score Range**: CHECK constraints enforce 0-100 on ai_score and manual_score
- **Order Uniqueness**: UNIQUE constraint on (test_id, order_num) for questions
- **Answer Uniqueness**: UNIQUE constraint on (submission_id, question_id) for answers
- **Review Uniqueness**: UNIQUE constraint on answer_id (one review per answer)

---

## Migration Strategy

### Initial Migrations

**001_create_users.up.sql**:
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK(role IN ('creator', 'reviewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX users_email_idx ON users(email);
CREATE INDEX users_role_idx ON users(role);
```

**002_create_tests.up.sql**:
```sql
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

CREATE INDEX tests_creator_id_idx ON tests(creator_id);
CREATE INDEX tests_published_idx ON tests(is_published) WHERE is_published = true;
```

**003_create_questions.up.sql**:
```sql
CREATE TABLE questions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    test_id UUID NOT NULL REFERENCES tests(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    expected_answer TEXT NOT NULL,
    order_num INTEGER NOT NULL CHECK(order_num > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(test_id, order_num)
);

CREATE INDEX questions_test_id_order_idx ON questions(test_id, order_num);
```

**004_create_submissions.up.sql**:
```sql
CREATE TABLE submissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    test_id UUID NOT NULL REFERENCES tests(id),
    access_email VARCHAR(255) NOT NULL,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ai_total_score NUMERIC(5,2),
    manual_total_score NUMERIC(5,2)
);

CREATE INDEX submissions_test_id_idx ON submissions(test_id);
CREATE INDEX submissions_email_idx ON submissions(test_id, access_email);
```

**005_create_answers.up.sql**:
```sql
CREATE TABLE answers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions(id),
    text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(submission_id, question_id)
);

CREATE INDEX answers_submission_id_idx ON answers(submission_id);
CREATE INDEX answers_question_id_idx ON answers(question_id);
```

**006_create_reviews.up.sql**:
```sql
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

CREATE UNIQUE INDEX reviews_answer_id_idx ON reviews(answer_id);
CREATE INDEX reviews_reviewer_id_idx ON reviews(reviewer_id);
```

### Down Migrations

Each migration has corresponding `.down.sql`:
```sql
-- 006_create_reviews.down.sql
DROP TABLE IF EXISTS reviews;

-- 005_create_answers.down.sql
DROP TABLE IF EXISTS answers;

-- 004_create_submissions.down.sql
DROP TABLE IF EXISTS submissions;

-- 003_create_questions.down.sql
DROP TABLE IF EXISTS questions;

-- 002_create_tests.down.sql
DROP TABLE IF EXISTS tests;

-- 001_create_users.down.sql
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS "uuid-ossp";
```

---

## Sample Data Scenarios

### Scenario 1: New Test with Questions

```sql
-- Creator creates test
INSERT INTO tests (id, creator_id, title, description, allow_retakes, is_published)
VALUES ('test-uuid', 'creator-uuid', 'Essay Writing 101', 'Write thoughtful essays', false, true);

-- Add questions
INSERT INTO questions (id, test_id, text, expected_answer, order_num)
VALUES 
  ('q1-uuid', 'test-uuid', 'What is Clean Architecture?', 'Separation of concerns...', 1),
  ('q2-uuid', 'test-uuid', 'Explain SOLID principles', 'Five principles: SRP...', 2);
```

### Scenario 2: Participant Submission

```sql
-- Participant submits (anonymous)
INSERT INTO submissions (id, test_id, access_email, submitted_at)
VALUES ('sub-uuid', 'test-uuid', 'student@example.com', NOW());

-- Participant's answers
INSERT INTO answers (id, submission_id, question_id, text)
VALUES 
  ('a1-uuid', 'sub-uuid', 'q1-uuid', 'Clean Architecture separates business logic...'),
  ('a2-uuid', 'sub-uuid', 'q2-uuid', 'SOLID stands for Single Responsibility...');
```

### Scenario 3: AI Scoring

```sql
-- AI worker scores each answer
INSERT INTO reviews (id, answer_id, ai_score, ai_feedback, ai_scored_at)
VALUES 
  ('r1-uuid', 'a1-uuid', 85.5, 'Good explanation of separation of concerns', NOW()),
  ('r2-uuid', 'a2-uuid', 78.0, 'Correct but missing Interface Segregation details', NOW());

-- Update submission total
UPDATE submissions
SET ai_total_score = 163.5
WHERE id = 'sub-uuid';
```

### Scenario 4: Manual Review Override

```sql
-- Reviewer adds manual scoring
UPDATE reviews
SET reviewer_id = 'reviewer-uuid',
    manual_score = 90.0,
    manual_feedback = 'Excellent work, minor grammar issues',
    manual_scored_at = NOW()
WHERE answer_id = 'a1-uuid';

-- Update submission manual total
UPDATE submissions
SET manual_total_score = 90.0
WHERE id = 'sub-uuid';
```

---

## Performance Considerations

### Indexing Strategy

**Query Patterns**:
- List user's tests: `SELECT * FROM tests WHERE creator_id = ? ORDER BY created_at DESC`
  - Index: `tests_creator_id_idx`
- Get test with questions: `SELECT * FROM questions WHERE test_id = ? ORDER BY order_num`
  - Index: `questions_test_id_order_idx`
- Get submission with answers and reviews:
  ```sql
  SELECT a.*, r.* 
  FROM answers a 
  LEFT JOIN reviews r ON r.answer_id = a.id
  WHERE a.submission_id = ?
  ```
  - Index: `answers_submission_id_idx`, `reviews_answer_id_idx` (unique)
- Check retake eligibility: `SELECT COUNT(*) FROM submissions WHERE test_id = ? AND access_email = ?`
  - Index: `submissions_email_idx` (composite)

### Query Optimization

1. **Avoid N+1**: Use JOINs or batch queries when fetching related entities
2. **Pagination**: Limit results for large lists (tests, submissions)
3. **Partial Indexes**: `tests_published_idx` only indexes published tests
4. **Denormalization**: Total scores in submissions table (avoid SUM on every read)

### Scaling Notes

- **Connection Pooling**: Use pgx connection pool (max 20 connections)
- **Read Replicas**: Future consideration for reviewer dashboard queries
- **Partitioning**: Not needed for baseline (expect < 100K rows per table)

---

## Validation Rules

### Domain Entities

**User**:
- Email: Valid format (RFC 5322), max 255 chars
- Role: Enum ('creator', 'reviewer')
- Password: Min 8 chars, hashed with bcrypt cost 12

**Test**:
- Title: 1-255 chars
- Description: 0-5000 chars (optional)
- Must have at least 1 question to publish

**Question**:
- Text: 1-10,000 chars
- Expected Answer: 1-10,000 chars
- Order: Positive integer, unique within test

**Submission**:
- Access Email: Valid format, max 255 chars
- Must answer all questions in test

**Answer**:
- Text: 1-50,000 chars (essay length)
- Must reference existing question

**Review**:
- AI Score: 0-100 (database constraint)
- Manual Score: 0-100 (database constraint)
- At least one of (ai_score, manual_score) must be present

---

## Future Enhancements (Out of Scope for Baseline)

1. **Soft Deletes**: Add `deleted_at` to tests, questions (preserve data)
2. **Audit Log**: Track all changes to tests, questions (who/when/what)
3. **Rich Text**: Store formatted text (Markdown/HTML) for questions/answers
4. **Attachments**: File uploads for questions (images, PDFs)
5. **Question Banks**: Reusable question library across tests
6. **Test Templates**: Clone existing tests
7. **Time Limits**: `time_limit_minutes` on tests
8. **Question Types**: Multiple choice, short answer (not just essays)
9. **Partial Submissions**: Save draft answers before final submit
10. **Analytics**: Aggregate statistics (avg scores, completion rates)

---

## Summary

The data model supports all baseline requirements:
- ✅ User authentication (creators/reviewers) with role-based access
- ✅ Tests with multiple essay questions
- ✅ Anonymous participant submissions
- ✅ Configurable retake policy per test
- ✅ Per-answer AI scoring with feedback
- ✅ Manual review override capability
- ✅ Total score aggregation (AI and manual)
- ✅ Email-based access token flow (access_email field)

All entities map to feature requirements in spec.md. Database schema enforces data integrity with appropriate constraints, indexes for query performance, and relationships with proper cascade behavior.

**Ready for Phase 1 continuation**: contracts/openapi.yaml and quickstart.md.
