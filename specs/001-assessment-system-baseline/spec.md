# Feature Specification: Flexible Assessment System Baseline

**Feature Branch**: `001-assessment-system-baseline`  
**Created**: 2026-03-26  
**Status**: Draft  
**Input**: User description: "Flexible Assessment System - Platform untuk membuat dan mengelola ujian essay tanpa login untuk peserta, dilengkapi AI reviewer"

## Clarifications

### Session 2026-03-26

- Q: Should the system support password reset functionality for creators and reviewers? → A: Yes - email-based password reset flow (user requests reset, receives email with time-limited token, sets new password)
- Q: How should the system identify and handle duplicate submissions? → A: One submission per email per test by default, but creators can enable retakes which allows multiple submissions (keeps latest)
- Q: What information should be provided to the AI model for generating scores and feedback? → A: Question text + expected answer + participant answer (AI compares participant answer against expected answer and question context)
- Q: What should be the access control and visibility rules for tests and submissions? → A: Creators see only their own tests; Reviewers see all tests and submissions system-wide
- Q: Should manual reviewer scoring be per-answer or per-submission, and how should it relate to AI scoring? → A: Per-answer scoring - each answer gets individual score (0-100) and feedback, reviewers can override each answer's AI score independently

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Test Creator Journey (Priority: P1)

A test creator (dosen/guru/HR) needs to create an essay test and share it with participants without requiring them to have accounts.

**Why this priority**: This is the core value proposition - enabling test creation and distribution. Without this, the platform has no purpose.

**Independent Test**: Can be fully tested by creating a test with questions, generating an access link, and verifying the link is accessible. Delivers immediate value by enabling test distribution.

**Acceptance Scenarios**:

1. **Given** a registered creator is logged in, **When** they navigate to create test page and enter test details (title, description), **Then** a new test is created with a unique access token
2. **Given** a test exists, **When** the creator adds essay questions with expected answers, **Then** questions are saved in specified order and associated with the test
3. **Given** a test has questions, **When** the creator activates the test, **Then** a shareable access link is generated that participants can use
4. **Given** a creator is configuring a test, **When** they enable the retake option, **Then** participants can submit multiple times and the latest submission is considered final
5. **Given** a test is active, **When** the creator wants to stop accepting submissions, **Then** they can deactivate the test and the access link becomes invalid

---

### User Story 2 - Participant Submission Journey (Priority: P1)

A participant receives a test link, enters their information, completes the essay questions, and submits their answers - all without creating an account.

**Why this priority**: This is the second half of the core value - frictionless participation. Both test creation and submission must work for an MVP.

**Independent Test**: Can be fully tested by accessing a test link, entering participant details, answering questions, and submitting. Delivers value by enabling actual test-taking.

**Acceptance Scenarios**:

1. **Given** a participant receives an active test link, **When** they open the link, **Then** they see a form to enter their name and email
2. **Given** a participant has entered their details, **When** they proceed, **Then** they see all essay questions for the test in order
3. **Given** a participant is viewing questions, **When** they type answers into text areas, **Then** their answers are captured for each question
4. **Given** a participant has answered questions, **When** they submit, **Then** a submission record is created with all answers and marked as 'submitted' status
5. **Given** a participant submits answers to a test with retakes disabled, **When** submission is successful, **Then** they see a confirmation message and cannot submit again with the same email
6. **Given** a participant submits answers to a test with retakes enabled, **When** they access the test link again with the same email, **Then** they can submit again and the new submission replaces the previous one

---

### User Story 3 - AI-Assisted Review (Priority: P2)

Test creators want AI to automatically score and provide feedback on essay answers to reduce manual review workload.

**Why this priority**: This is a key differentiator but not essential for MVP functionality. Manual review is still possible without it.

**Independent Test**: Can be fully tested by submitting an answer to an AI-enabled question and verifying that AI score and feedback are generated asynchronously. Delivers value by reducing review time.

**Acceptance Scenarios**:

1. **Given** a creator is adding a question, **When** they enable AI review for that question, **Then** the question is marked as AI-enabled
2. **Given** a participant submits answers to an AI-enabled question, **When** the submission is processed, **Then** the answer is queued for AI scoring asynchronously
3. **Given** an answer is queued for AI scoring, **When** the AI service processes it with the question text, expected answer, and participant answer, **Then** an AI score (0-100) and feedback text are stored with the answer
4. **Given** AI scoring fails for any reason, **When** the failure is detected, **Then** the answer remains without AI score/feedback and does not block manual review

---

### User Story 4 - Manual Review and Override (Priority: P2)

Reviewers need to view all submissions, see AI-generated scores/feedback, and provide or override with their own assessment.

**Why this priority**: Essential for quality control but can be deferred if focusing purely on test creation/taking first. Works independently of AI review.

**Independent Test**: Can be fully tested by logging in as reviewer, viewing submissions, and adding/editing scores and feedback. Delivers value by enabling human quality assurance.

**Acceptance Scenarios**:

1. **Given** a reviewer is logged in, **When** they navigate to submissions page, **Then** they see all submissions across all tests system-wide with participant info and status
2. **Given** a reviewer is logged in, **When** they filter submissions by test, **Then** they can see submissions for any test in the system regardless of who created it
3. **Given** a reviewer selects a submission, **When** viewing submission details, **Then** they see all questions with participant answers and any AI-generated scores/feedback for each answer
4. **Given** a reviewer is viewing an individual answer, **When** they enter a score (0-100) and feedback text for that answer, **Then** their manual review is saved for that specific answer
5. **Given** an answer has both AI and manual scores, **When** displaying the answer, **Then** the manual score takes precedence as the final score for that answer
6. **Given** a reviewer has reviewed all answers in a submission, **When** they mark the submission as reviewed, **Then** the submission status changes to 'reviewed'

---

### User Story 5 - User Authentication and Authorization (Priority: P1)

Test creators and reviewers need secure accounts to manage tests and reviews, while participants should access tests without authentication.

**Why this priority**: Security is foundational and must be in place from the start. Registration/login for creators is essential for data ownership.

**Independent Test**: Can be fully tested by registering a user, logging in, verifying access to protected resources, and logging out. Delivers value by enabling secure multi-user operation.

**Acceptance Scenarios**:

1. **Given** a new user wants to create tests, **When** they register with name, email, and password, **Then** a new user account is created with creator or reviewer role
2. **Given** a registered user has valid credentials, **When** they log in with email and password, **Then** they receive a JWT token for authenticated requests
3. **Given** an authenticated user has a valid token, **When** they access protected endpoints (create test, view submissions), **Then** they are authorized based on their role
4. **Given** a user is logged in, **When** they log out, **Then** their session is invalidated (client discards token)
5. **Given** a participant accesses a test link, **When** they view and submit the test, **Then** no authentication is required (token in link is sufficient)
6. **Given** a user forgot their password, **When** they request a password reset with their email, **Then** a time-limited reset token is sent to their email
7. **Given** a user receives a password reset email, **When** they click the reset link and enter a new password, **Then** their password is updated and they can log in with the new password

---

### User Story 6 - Test and Question Management (Priority: P1)

Creators need full CRUD capabilities for tests and questions to manage their assessments over time.

**Why this priority**: While test creation is P1, editing and deletion are important for mistake correction and lifecycle management. Essential for production use.

**Independent Test**: Can be fully tested by creating, editing, viewing, and deleting tests and questions. Delivers value by enabling iterative test refinement.

**Acceptance Scenarios**:

1. **Given** a creator is logged in, **When** they view the test list, **Then** they see only tests they have created
2. **Given** a creator has an existing test, **When** they edit the test details (title, description), **Then** the test is updated with new information
3. **Given** a test has questions, **When** the creator edits a question (text, expected answer, AI settings), **Then** the question is updated
4. **Given** a test has multiple questions, **When** the creator reorders questions, **Then** the order_number is updated and questions display in new order
5. **Given** a creator wants to remove a question, **When** they delete it, **Then** the question is soft-deleted (deleted_at set) and no longer appears in the test
6. **Given** a creator wants to archive a test, **When** they delete the test, **Then** the test and all its questions are soft-deleted but data is preserved
7. **Given** a creator tries to access another creator's test, **When** they attempt to view or edit it, **Then** they receive an authorization error

---

### Edge Cases

- What happens when a participant tries to access a deactivated test? System should display a friendly error message that the test is no longer available.
- What happens when a participant tries to access a test with an invalid token? System should return a 404 or error page indicating the test doesn't exist.
- What happens when AI scoring service is unavailable or times out? System should log the error, leave answer without AI score, and allow manual review to proceed.
- What happens when multiple participants submit at the exact same time? System should handle concurrent submissions independently with proper database transaction isolation.
- What happens when a creator deletes a test that has existing submissions? Test and questions are soft-deleted, but submissions remain accessible for reviewers (referential integrity maintained).
- What happens when a reviewer is also a creator? User should be able to perform both roles - as creator they see only their own tests, as reviewer they see all tests system-wide.
- What happens when a creator tries to access another creator's test? System should return an authorization error (403 Forbidden) preventing access.
- What happens when participant email is duplicated across different submissions? If same email for same test: allowed only if retakes enabled (replaces previous submission). If same email for different tests: always allowed.
- What happens when a participant tries to resubmit to a test with retakes disabled? System should display an error message indicating they have already submitted and retakes are not allowed.
- What happens when a question's expected answer is very long? System should use TEXT field type to support lengthy expected answers.
- What happens when AI returns invalid or malformed scoring data? System should validate AI response, log errors, and skip updating score/feedback if invalid.
- What happens when a password reset token expires or is already used? System should display a clear error message and offer to send a new reset email.
- What happens when a reviewer only scores some answers in a submission but not all? Submission can remain in 'submitted' status until reviewer marks it 'reviewed'; reviewers can partially review and return later.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow users to register as creators or reviewers with name, email, and password
- **FR-002**: System MUST validate email addresses are unique and properly formatted
- **FR-003**: System MUST authenticate users via JWT tokens for protected operations
- **FR-004**: System MUST allow users to request password reset via email
- **FR-005**: System MUST generate time-limited password reset tokens and send them via email
- **FR-006**: System MUST allow users to set a new password using a valid reset token
- **FR-007**: System MUST invalidate password reset tokens after use or expiration
- **FR-008**: System MUST allow creators to create tests with title and optional description
- **FR-009**: System MUST generate unique, secure access tokens for each test
- **FR-010**: System MUST allow creators to add essay questions to tests with question text, expected answer, and AI-enable flag
- **FR-011**: System MUST allow creators to set and modify the order of questions within a test
- **FR-012**: System MUST allow creators to activate and deactivate tests
- **FR-013**: System MUST allow creators to enable or disable retakes for a test
- **FR-014**: System MUST allow participants to access active tests via tokenized link without authentication
- **FR-015**: System MUST collect participant name and email before showing test questions
- **FR-016**: System MUST display all questions in specified order to participants
- **FR-017**: System MUST allow participants to submit essay answers for each question
- **FR-018**: System MUST create submission records with participant info and all answers upon submission
- **FR-019**: System MUST prevent duplicate submissions from the same participant email for the same test when retakes are disabled
- **FR-020**: System MUST allow multiple submissions from the same participant email when retakes are enabled, keeping only the latest submission
- **FR-021**: System MUST asynchronously queue AI-enabled questions for scoring after submission
- **FR-022**: System MUST integrate with Groq AI model to generate scores (0-100) and feedback for answers by providing question text, expected answer, and participant answer
- **FR-023**: System MUST store AI-generated scores and feedback with the corresponding answers
- **FR-024**: System MUST allow creators to view and manage only tests they have created
- **FR-025**: System MUST prevent creators from accessing or modifying tests created by other creators
- **FR-026**: System MUST allow reviewers to view all tests and submissions system-wide regardless of creator
- **FR-027**: System MUST allow reviewers to view detailed submission data including all questions, participant answers, and AI scores/feedback for each answer
- **FR-028**: System MUST allow reviewers to input or override score (0-100) and feedback text for each individual answer
- **FR-029**: System MUST update submission status to 'reviewed' when reviewer completes assessment
- **FR-030**: System MUST prioritize manual reviewer scores over AI scores for each answer when both exist
- **FR-031**: System MUST allow creators to edit existing tests and questions they own
- **FR-032**: System MUST allow creators to delete tests and questions they own (soft delete)
- **FR-033**: System MUST record created_at and updated_at timestamps for all entities
- **FR-034**: System MUST support soft deletion (deleted_at) for users, tests, questions, submissions, answers, and reviews
- **FR-035**: System MUST maintain referential integrity between related entities (tests→questions, submissions→answers, etc.)
- **FR-036**: System MUST log all significant operations with structured logging
- **FR-037**: System MUST expose metrics for monitoring (Prometheus format)
- **FR-038**: System MUST support distributed tracing for request flows (OpenTelemetry)

### Key Entities *(include if feature involves data)*

- **User**: Represents creators and reviewers; attributes include name, email, password (hashed), role (creator/reviewer); identified by unique email
- **Test**: Represents an assessment; attributes include title, description, access_token (unique), is_active (boolean), allow_retakes (boolean, default false), created_by (user reference); owned by a creator
- **Question**: Represents an essay question within a test; attributes include question_text, type (essay), expected_answer, ai_enabled (boolean), order_number; belongs to a test
- **Submission**: Represents a participant's test attempt; attributes include test reference, participant_name, participant_email, status (submitted/reviewed), submitted_at; contains multiple answers
- **Answer**: Represents a response to a question; attributes include submission reference, question reference, answer_text, ai_score (0-100), ai_feedback; links submission to question
- **Review**: Represents manual per-answer assessment by a reviewer; attributes include answer reference, reviewer reference, score (0-100), feedback; provides human override of AI scores on individual answers

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Creators can create a complete test with multiple questions and generate a shareable link in under 5 minutes
- **SC-002**: Participants can access a test link, complete their submission, and receive confirmation in under 10 minutes (for a 5-question test)
- **SC-003**: System handles at least 100 concurrent participant submissions without response time degradation beyond 2 seconds for non-AI endpoints
- **SC-004**: AI scoring completes within 30 seconds for 90% of submissions (not blocking user submission flow)
- **SC-005**: Reviewers can view and manually score a submission in under 3 minutes
- **SC-006**: 95% of API requests (excluding AI processing) complete in under 2 seconds
- **SC-007**: System maintains 99% uptime for test access and submission endpoints
- **SC-008**: Zero data loss for submissions and answers (all submissions are persisted successfully)
- **SC-009**: Automated tests achieve minimum 70% code coverage across all modules
- **SC-010**: System successfully processes and queues AI scoring asynchronously without blocking participant submission confirmation

## Assumptions

- Users (creators/reviewers) have stable internet connectivity to access the web platform
- Participants have basic web browser capabilities (modern browsers with JavaScript enabled)
- Groq AI API is available and accessible from the backend infrastructure
- Message broker infrastructure (RabbitMQ/Redis/SQS) is available for async AI processing
- PostgreSQL database is available and properly configured for production use
- Test creators will manage participant access through the link distribution (no built-in email invitation system in v1)
- Participants are expected to complete tests in a single session (no save-and-resume feature in baseline)
- Email validation is sufficient for participant identity (no email verification for participants)
- Creators/reviewers will use strong passwords (basic password strength validation, no complex policy enforcement in v1)
- The system will operate in a single timezone (UTC) for all timestamps
- File uploads for questions/answers are out of scope for baseline (text-only essay questions)
- Real-time collaboration features (multiple reviewers scoring simultaneously) are out of scope
- Plagiarism detection is out of scope for baseline (future enhancement)
- Participant test analytics/insights (time spent, keystrokes) are out of scope for baseline
- Mobile app versions are out of scope (responsive web interface only)
- Multi-language support (i18n) is out of scope for baseline
- Payment/subscription features are out of scope for baseline (free platform)
- Advanced test features (time limits, randomized questions, question banks) are out of scope for baseline
