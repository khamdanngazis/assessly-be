# Tasks: Flexible Assessment System Baseline

**Feature**: 001-assessment-system-baseline  
**Input**: Design documents from `/specs/001-assessment-system-baseline/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/openapi.yaml, quickstart.md

**Generated**: 2026-03-26

## Format: `- [ ] [TaskID] [P?] [Story?] Description with file path`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: User story label (US1-US6) for tracking which story this task belongs to
- **File paths**: Exact paths within `assessly-be/` repository

---

## Phase 1: Setup (Project Initialization)

**Purpose**: Bootstrap Go project structure and tooling

- [X] T001 Initialize Go module with `go mod init github.com/<org>/assessly-be` at repository root
- [X] T002 Create directory structure: cmd/{api,worker}/, internal/{domain,usecase,delivery,infrastructure}/, migrations/, tests/{unit,integration,contract}/
- [X] T003 [P] Create .env.example with all required environment variables per quickstart.md
- [X] T004 [P] Create .gitignore for Go (bin/, .env, coverage.out, vendor/)
- [X] T005 [P] Create Makefile with targets: build, test, migrate-up, migrate-down, run-api, run-worker, docker-up
- [X] T006 [P] Add go.mod dependencies: chi, pgx, golang-jwt/jwt, go-redis, gomail, testify, golang-migrate
- [X] T007 [P] Create README.md linking to quickstart.md and architecture overview

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story implementation

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Database Setup

- [X] T008 Create migration 001_create_users.up.sql in migrations/ with users table schema per data-model.md
- [X] T009 Create migration 001_create_users.down.sql in migrations/ to drop users table
- [X] T010 [P] Create migration 002_create_tests.up.sql in migrations/ with tests table schema per data-model.md
- [X] T011 [P] Create migration 002_create_tests.down.sql in migrations/ to drop tests table
- [X] T012 [P] Create migration 003_create_questions.up.sql in migrations/ with questions table schema per data-model.md
- [X] T013 [P] Create migration 003_create_questions.down.sql in migrations/ to drop questions table
- [X] T014 [P] Create migration 004_create_submissions.up.sql in migrations/ with submissions table schema per data-model.md
- [X] T015 [P] Create migration 004_create_submissions.down.sql in migrations/ to drop submissions table
- [X] T016 [P] Create migration 005_create_answers.up.sql in migrations/ with answers table schema per data-model.md
- [X] T017 [P] Create migration 005_create_answers.down.sql in migrations/ to drop answers table
- [X] T018 [P] Create migration 006_create_reviews.up.sql in migrations/ with reviews table schema per data-model.md
- [X] T019 [P] Create migration 006_create_reviews.down.sql in migrations/ to drop reviews table

### Configuration & Infrastructure Base

- [X] T020 Create configuration loader in internal/infrastructure/config/config.go that loads ENV variables per quickstart.md
- [X] T021 Create database connection setup in internal/infrastructure/postgres/db.go with pgx connection pool
- [X] T022 [P] Create Redis connection setup in internal/infrastructure/redis/client.go with go-redis client
- [X] T023 [P] Create structured logger setup in internal/infrastructure/logging/logger.go using slog
- [X] T024 Create migration runner in cmd/api/main.go that auto-migrates on startup using golang-migrate

### Domain Layer (Core Entities & Interfaces)

- [X] T025 [P] Create User entity in internal/domain/user.go with ID, Email, PasswordHash, Role, CreatedAt, UpdatedAt
- [X] T026 [P] Create Test entity in internal/domain/test.go with ID, CreatorID, Title, Description, AllowRetakes, IsPublished, CreatedAt, UpdatedAt
- [X] T027 [P] Create Question entity in internal/domain/question.go with ID, TestID, Text, ExpectedAnswer, OrderNum, CreatedAt
- [X] T028 [P] Create Submission entity in internal/domain/submission.go with ID, TestID, AccessEmail, SubmittedAt, AITotalScore, ManualTotalScore
- [X] T029 [P] Create Answer entity in internal/domain/answer.go with ID, SubmissionID, QuestionID, Text, CreatedAt
- [X] T030 [P] Create Review entity in internal/domain/review.go with ID, AnswerID, ReviewerID, AIScore, AIFeedback, AIScoredat, ManualScore, ManualFeedback, ManualScoredAt
- [X] T031 [P] Create domain errors in internal/domain/errors.go (ErrNotFound, ErrUnauthorized, ErrValidation, ErrConflict)
- [X] T032 Create repository interfaces in internal/domain/repository.go (UserRepository, TestRepository, QuestionRepository, SubmissionRepository, AnswerRepository, ReviewRepository)

### API Foundation

- [X] T033 Create chi router setup in internal/delivery/http/router.go with /api/v1 versioning
- [X] T034 [P] Create recovery middleware in internal/delivery/http/middleware/recovery.go
- [X] T035 [P] Create logging middleware in internal/delivery/http/middleware/logging.go using slog
- [X] T036 [P] Create CORS middleware in internal/delivery/http/middleware/cors.go
- [X] T037 Create health check handler in internal/delivery/http/health_handler.go (GET /health) checking DB and Redis
- [X] T038 Create API server entrypoint in cmd/api/main.go with graceful shutdown per plan.md

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 5 - User Authentication and Authorization (Priority: P1) 🎯

**Goal**: Secure creator/reviewer accounts with JWT authentication and password reset flow

**Independent Test**: Register user → Login → Access protected endpoint → Logout. Password reset: Request → Receive email → Reset with token.

### Implementation for User Story 5

- [X] T039 [P] [US5] Implement UserRepository in internal/infrastructure/postgres/user_repository.go (Create, FindByEmail, FindByID, Update methods)
- [X] T040 [P] [US5] Create JWT service in internal/infrastructure/auth/jwt.go (GenerateToken, ValidateToken, ExtractClaims)
- [X] T041 [P] [US5] Create password hasher in internal/infrastructure/auth/password.go using bcrypt (Hash, Compare)
- [X] T042 [P] [US5] Create email sender in internal/infrastructure/email/smtp.go using gomail per research.md TD-002
- [X] T043 [US5] Create RegisterUser use case in internal/usecase/auth/register.go (validates email uniqueness, hashes password, creates user) - depends on T039, T041
- [X] T044 [US5] Create LoginUser use case in internal/usecase/auth/login.go (validates credentials, generates JWT) - depends on T039, T040, T041
- [X] T045 [P] [US5] Create RequestPasswordReset use case in internal/usecase/auth/request_reset.go (generates reset token, sends email) - depends on T039, T040, T042
- [X] T046 [P] [US5] Create ResetPassword use case in internal/usecase/auth/reset_password.go (validates token, updates password) - depends on T039, T040, T041
- [X] T047 [US5] Create JWT authentication middleware in internal/delivery/http/middleware/auth.go (validates Bearer token, extracts user context)
- [X] T048 [US5] Create auth handlers in internal/delivery/http/auth_handler.go with POST /api/v1/auth/register, POST /api/v1/auth/login, POST /api/v1/auth/reset-password, PUT /api/v1/auth/reset-password/:token per openapi.yaml
- [X] T049 [US5] Add auth routes to router in internal/delivery/http/router.go

**Checkpoint**: Authentication complete - creators/reviewers can register, login, reset passwords. JWT middleware ready for protected endpoints.

---

## Phase 4: User Story 1 - Test Creator Journey (Priority: P1) 🎯 MVP Core

**Goal**: Creators can create tests, add questions, and publish for participant access

**Independent Test**: Login as creator → Create test → Add 2+ questions → Publish test → Verify access link works

### Implementation for User Story 1

- [X] T050 [P] [US1] Implement TestRepository in internal/infrastructure/postgres/test_repository.go (Create, FindByID, FindByCreatorID, Update, Delete, Publish methods)
- [X] T051 [P] [US1] Implement QuestionRepository in internal/infrastructure/postgres/question_repository.go (Create, FindByID, FindByTestID, Update, Delete methods)
- [X] T052 [US1] Create CreateTest use case in internal/usecase/test/create_test.go (validates title, creates test as draft) - depends on T050
- [X] T053 [US1] Create AddQuestion use case in internal/usecase/test/add_question.go (validates text/expected_answer, ensures unique order_num) - depends on T051
- [X] T054 [US1] Create PublishTest use case in internal/usecase/test/publish_test.go (validates >= 1 question exists, sets is_published=true) - depends on T050, T051
- [X] T055 [US1] Create test handlers in internal/delivery/http/test_handler.go with POST /api/v1/tests, GET /api/v1/tests/:id, POST /api/v1/tests/:id/publish per openapi.yaml
- [X] T056 [US1] Create question handlers in internal/delivery/http/question_handler.go with POST /api/v1/tests/:testId/questions, GET /api/v1/tests/:testId/questions per openapi.yaml
- [X] T057 [US1] Add test and question routes to router with JWT auth middleware

**Checkpoint**: Creators can create, configure, and publish tests. Foundation for participant submission ready.

---

## Phase 5: User Story 6 - Test and Question Management (Priority: P1) 🎯

**Goal**: Full CRUD capabilities for tests and questions with ownership validation

**Independent Test**: Create test → Edit title/description → Reorder questions → Delete question → Delete test. Verify ownership (cannot access other creator's tests).

### Implementation for User Story 6

- [ ] T058 [US6] Create ListTests use case in internal/usecase/test/list_tests.go (filters by creator_id for creators, all published for reviewers)
- [ ] T059 [P] [US6] Create GetTest use case in internal/usecase/test/get_test.go (validates ownership for creators, allows all for reviewers) - depends on T050
- [ ] T060 [P] [US6] Create UpdateTest use case in internal/usecase/test/update_test.go (validates ownership, updates title/description/allow_retakes) - depends on T050
- [ ] T061 [P] [US6] Create DeleteTest use case in internal/usecase/test/delete_test.go (validates ownership, soft delete) - depends on T050
- [ ] T062 [P] [US6] Create UnpublishTest use case in internal/usecase/test/unpublish_test.go (validates ownership, sets is_published=false) - depends on T050
- [ ] T063 [P] [US6] Create UpdateQuestion use case in internal/usecase/test/update_question.go (validates test ownership, updates question) - depends on T051
- [ ] T064 [P] [US6] Create DeleteQuestion use case in internal/usecase/test/delete_question.go (validates test ownership, soft delete) - depends on T051
- [ ] T065 [US6] Add handlers to internal/delivery/http/test_handler.go: GET /api/v1/tests, PUT /api/v1/tests/:id, DELETE /api/v1/tests/:id, POST /api/v1/tests/:id/unpublish
- [ ] T066 [US6] Add handlers to internal/delivery/http/question_handler.go: GET /api/v1/tests/:testId/questions/:id, PUT /api/v1/tests/:testId/questions/:id, DELETE /api/v1/tests/:testId/questions/:id
- [ ] T067 [US6] Add new routes to router with JWT auth middleware

**Checkpoint**: Full test/question lifecycle management complete. Creators have complete control over their content.

---

## Phase 6: User Story 2 - Participant Submission Journey (Priority: P1) 🎯 MVP Core

**Goal**: Anonymous participants can access tests via link, submit answers without authentication

**Independent Test**: Request test access token → Receive email → Open test link → Submit answers → Poll for results

### Implementation for User Story 2

- [X] T068 [P] [US2] Implement SubmissionRepository in internal/infrastructure/postgres/submission_repository.go (Create, FindByID, FindByTestID, CountByTestAndEmail methods)
- [X] T069 [P] [US2] Implement AnswerRepository in internal/infrastructure/postgres/answer_repository.go (Create, FindBySubmissionID, FindByAnswerID methods)
- [X] T070 [US2] Create GenerateAccessToken use case in internal/usecase/submission/generate_access_token.go (validates test published, generates JWT with test_id+email, sends via email) - depends on T040, T042, T050
- [X] T071 [US2] Create SubmitTest use case in internal/usecase/submission/submit_test.go (validates access token, checks retake policy, creates submission+answers) - depends on T040, T068, T069, T050, T051
- [X] T072 [US2] Create GetSubmission use case in internal/usecase/submission/get_submission.go (validates access token or reviewer role, returns submission with answers and reviews)
- [X] T073 [US2] Create submission handlers in internal/delivery/http/submission_handler.go with POST /api/v1/submissions/access, POST /api/v1/submissions, GET /api/v1/submissions/:id per openapi.yaml
- [X] T074 [US2] Add submission routes to router (access and submit are public, get requires auth or access token)

**Checkpoint**: Participants can access and submit tests without accounts. Core submission flow complete.

---

## Phase 7: User Story 3 - AI-Assisted Review (Priority: P2)

**Goal**: Automatic AI scoring using Groq API with async processing via Redis Streams

**Independent Test**: Submit test answers → Verify queued in Redis → Worker processes → AI score/feedback stored → Participant sees results

### Implementation for User Story 3

- [ ] T075 [P] [US3] Implement ReviewRepository in internal/infrastructure/postgres/review_repository.go (Create, FindByAnswerID, Update, UpsertAIScore methods)
- [ ] T076 [P] [US3] Create Redis queue client in internal/infrastructure/redis/queue.go with Enqueue/Dequeue methods using XADD/XREADGROUP per research.md TD-001
- [ ] T077 [P] [US3] Create Groq AI client in internal/infrastructure/groq/client.go with ScoreAnswer(question, expected, answer) method per research.md TD-006
- [ ] T078 [US3] Create QueueAIScoring use case in internal/usecase/scoring/queue_scoring.go (enqueues submission_id to Redis after submission) - depends on T076
- [ ] T079 [US3] Create ScoreWithAI use case in internal/usecase/scoring/score_with_ai.go (calls Groq API, stores ai_score/ai_feedback, updates submission total) - depends on T075, T077
- [ ] T080 [US3] Modify SubmitTest use case to call QueueAIScoring after creating submission - update internal/usecase/submission/submit_test.go
- [ ] T081 [US3] Create worker consumer in internal/delivery/worker/scoring_consumer.go that dequeues from Redis and calls ScoreWithAI use case
- [ ] T082 [US3] Create worker entrypoint in cmd/worker/main.go with graceful shutdown
- [ ] T083 [US3] Update GetSubmission use case to include reviews with AI scores in response

**Checkpoint**: AI scoring operational. Submissions automatically queued and scored asynchronously.

---

## Phase 8: User Story 4 - Manual Review and Override (Priority: P2)

**Goal**: Reviewers can view all submissions and add/override AI scores per answer

**Independent Test**: Login as reviewer → View all submissions → Select submission → Add manual score/feedback to answers → Verify manual score overrides AI

### Implementation for User Story 4

- [ ] T084 [US4] Create ListSubmissions use case in internal/usecase/review/list_submissions.go (filters by test_id, accessible to all reviewers)
- [ ] T085 [US4] Create AddManualReview use case in internal/usecase/review/add_manual_review.go (validates reviewer role, upserts manual_score/manual_feedback, updates submission total) - depends on T075
- [ ] T086 [US4] Create GetReview use case in internal/usecase/review/get_review.go (returns review with AI and manual scores for an answer)
- [ ] T087 [US4] Create review handlers in internal/delivery/http/review_handler.go with PUT /api/v1/reviews/:answerId, GET /api/v1/reviews/:answerId per openapi.yaml
- [ ] T088 [US4] Create submission list handler: GET /api/v1/tests/:testId/submissions in internal/delivery/http/submission_handler.go
- [ ] T089 [US4] Add review routes to router with JWT auth middleware (requires reviewer role)
- [ ] T090 [US4] Update GetSubmission response to prioritize manual_score over ai_score when displaying results

**Checkpoint**: Manual review complete. Reviewers can oversee and adjust AI assessments.

---

## Final Phase: Testing, Deployment & Polish

**Purpose**: Comprehensive testing, containerization, CI/CD, and cross-cutting concerns

### Unit Tests (80%+ coverage for domain/usecase)

- [ ] T091 [P] Write unit tests for all auth use cases in tests/unit/auth_test.go
- [ ] T092 [P] Write unit tests for all test management use cases in tests/unit/test_test.go
- [ ] T093 [P] Write unit tests for all submission use cases in tests/unit/submission_test.go
- [ ] T094 [P] Write unit tests for all scoring use cases in tests/unit/scoring_test.go
- [ ] T095 [P] Write unit tests for all review use cases in tests/unit/review_test.go
- [ ] T096 [P] Write unit tests for JWT service in tests/unit/jwt_test.go
- [ ] T097 [P] Write unit tests for password hasher in tests/unit/password_test.go

### Integration Tests (testcontainers for DB/Redis)

- [X] T098 [P] Write integration test for auth flow (register→login→reset) in tests/integration/auth_integration_test.go using testcontainers
- [X] T099 [P] Write integration test for test creation and publishing flow in tests/integration/test_integration_test.go
- [X] T100 [P] Write integration test for participant submission flow in tests/integration/submission_integration_test.go
- [X] T101 [P] Write integration test for AI scoring worker flow in tests/integration/scoring_integration_test.go
- [X] T102 [P] Write integration test for manual review flow in tests/integration/review_integration_test.go

### Contract Tests (OpenAPI validation)

- [X] T103 Write contract test validator in tests/contract/openapi_test.go that validates all API responses against contracts/openapi.yaml
- [X] T104 [P] Write contract tests for auth endpoints in tests/contract/auth_contract_test.go
- [X] T105 [P] Write contract tests for test endpoints in tests/contract/test_contract_test.go
- [X] T106 [P] Write contract tests for submission endpoints in tests/contract/submission_contract_test.go
- [X] T107 [P] Write contract tests for review endpoints in tests/contract/review_contract_test.go

### Deployment & Infrastructure

- [X] T108 [P] Create Dockerfile for API server with multi-stage build per quickstart.md
- [X] T109 [P] Create Dockerfile for worker with multi-stage build
- [X] T110 Create docker-compose.yml orchestrating api and worker services (PostgreSQL and Redis run on host machine)
- [X] T111 [P] Create .dockerignore file
- [X] T112 Create GitHub Actions CI pipeline in .github/workflows/ci.yml (lint, test, build, security scan) per plan.md
- [X] T113 [P] Add golangci-lint configuration in .golangci.yml
- [ ] T114 [P] Add gosec security scanning to CI pipeline

### Observability & Monitoring

- [ ] T115 [P] Add Prometheus metrics endpoint in internal/delivery/http/metrics_handler.go (GET /metrics)
- [ ] T116 [P] Add request duration metrics in logging middleware
- [ ] T117 [P] Add error rate metrics in error handling middleware
- [ ] T118 [P] Add AI scoring duration metrics in scoring use case
- [ ] T119 [P] Add Redis queue depth metrics in worker consumer
- [ ] T120 [P] Configure OpenTelemetry tracing in cmd/api/main.go and cmd/worker/main.go per plan.md

### Documentation & Polish

- [ ] T121 [P] Add inline documentation for all public interfaces and complex functions
- [ ] T122 [P] Create ADR for message broker selection in docs/adr/001-message-broker.md
- [ ] T123 [P] Create ADR for auth strategy in docs/adr/002-authentication.md
- [ ] T124 [P] Create ADR for AI integration in docs/adr/003-groq-integration.md
- [ ] T125 [P] Update README.md with architecture diagram and getting started guide
- [ ] T126 Validate all success criteria from spec.md (SC-001 through SC-010)
- [ ] T127 Run full test suite and verify 70%+ coverage (`go test -cover ./...`)

---

## Implementation Strategy

### MVP Scope (Quick Value - Recommended First PR)

Focus on P1 stories only for initial release:
- **Phase 1-2**: Setup + Foundational (T001-T038)
- **Phase 3**: Authentication (T039-T049)
- **Phase 4**: Test Creator (T050-T057)
- **Phase 6**: Participant Submission (T068-T074)

This delivers core value: "Creators can create tests, participants can take them, results are stored."

### Parallel Execution Opportunities

**After Phase 2 complete**, these can run in parallel:
1. **Team A**: Authentication (US5) → Test Creator (US1) → Test Management (US6)
2. **Team B**: Participant Submission (US2)
3. **Team C**: AI Review (US3) → Manual Review (US4)

**Testing phase**: All unit/integration/contract tests can be written in parallel

### Dependency Graph

```
Phase 1-2 (Setup + Foundation)
    ↓
    ├─→ Phase 3 (US5: Auth) ──┬─→ Phase 4 (US1: Creator) ──→ Phase 5 (US6: Management)
    │                          │
    │                          └─→ Phase 6 (US2: Submission) ──→ Phase 7 (US3: AI Review)
    │                                                                     ↓
    │                                                               Phase 8 (US4: Manual Review)
    │
    └─→ Final Phase (Testing, Deployment, Polish) - can start after MVP delivered
```

### Task Count Summary

- **Total Tasks**: 127
- **Setup**: 7 tasks (T001-T007)
- **Foundational**: 31 tasks (T008-T038)
- **User Story 5** (Auth): 11 tasks (T039-T049)
- **User Story 1** (Creator): 8 tasks (T050-T057)
- **User Story 6** (Management): 10 tasks (T058-T067)
- **User Story 2** (Submission): 7 tasks (T068-T074)
- **User Story 3** (AI Review): 9 tasks (T075-T083)
- **User Story 4** (Manual Review): 7 tasks (T084-T090)
- **Testing & Deployment**: 37 tasks (T091-T127)

### Parallelization Summary

- **75 tasks marked [P]** can run in parallel within their phase
- **52 sequential tasks** have dependencies
- **Estimated Timeline**: 15 working days (3 weeks at 60% utilization per plan.md)

---

## Success Validation

After completing all tasks, verify these outcomes:

✅ **SC-001**: Creators create test + questions + publish in < 5 minutes  
✅ **SC-002**: Participants complete submission in < 10 minutes  
✅ **SC-003**: System handles 100+ concurrent submissions  
✅ **SC-004**: AI scoring completes in < 30 seconds for 90%  
✅ **SC-005**: Reviewers score submission in < 3 minutes  
✅ **SC-006**: 95% API requests complete in < 2 seconds  
✅ **SC-007**: 99% uptime maintained  
✅ **SC-008**: Zero data loss for submissions  
✅ **SC-009**: 70%+ test coverage achieved  
✅ **SC-010**: Async AI scoring doesn't block submissions  

---

**Generated**: 2026-03-26  
**Status**: Ready for implementation  
**Next Action**: Create feature branch and begin Phase 1 (Setup)
