# Implementation Plan: Flexible Assessment System Baseline

**Branch**: `001-assessment-system-baseline` | **Date**: 2026-03-26 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-assessment-system-baseline/spec.md`

**Note**: This plan covers the baseline implementation of the Flexible Assessment System - a platform for creating and managing essay exams with AI-assisted review.

## Summary

The Flexible Assessment System enables educators, HR professionals, and trainers to create essay-based tests that participants can access without accounts. The system provides AI-powered automatic scoring using Groq models with manual review capabilities. This baseline implementation establishes the core platform with user management, test/question CRUD, participant submission flow, async AI scoring, and reviewer workflows, all built on Clean Architecture principles using Go, PostgreSQL, and modern observability practices.

## Technical Context

**Language/Version**: Go (Golang) >= 1.21  
**Primary Dependencies**: 
- HTTP Router: gorilla/mux or chi (RESTful API)
- Database: lib/pq or pgx (PostgreSQL driver)
- JWT: golang-jwt/jwt
- HTTP Client: standard library net/http (Groq AI integration)
- Message Broker: To be selected (RabbitMQ/Redis Streams/AWS SQS)
- Migrations: pressly/goose or golang-migrate/migrate
- Observability: slog (structured logging), prometheus client, OpenTelemetry SDK
- Testing: testing package, testify/assert

**Storage**: PostgreSQL 14+ (transactions, JSONB support, full-text search for future)  
**Testing**: Go testing framework + testify, coverage target 70%+ (constitution requires 80% for critical paths)  
**Target Platform**: Linux server (containerized via Docker)  
**Project Type**: RESTful web service with async workers  
**Performance Goals**: 
- < 2s response time for 95% of non-AI API requests (SC-006)
- Support 100+ concurrent submissions without degradation (SC-003)
- AI scoring within 30 seconds for 90% of submissions (SC-004)

**Constraints**: 
- Clean Architecture mandatory (domain/usecase/delivery/infrastructure layers)
- SOLID principles non-negotiable
- 70% minimum test coverage, 80%+ for critical business logic
- All secrets via environment variables (never committed)
- Structured logging only (no sensitive data in logs)

**Scale/Scope**: 
- Multi-tenant (creators isolated, reviewers system-wide access)
- Async AI processing (non-blocking submission flow)
- Baseline version: single deployment instance acceptable
- Future: horizontal scaling support required (stateless design)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Architectural Gates

✅ **Clean Architecture (Principle I)**: 
- Feature REQUIRES domain/usecase/delivery/infrastructure separation
- Domain entities: User, Test, Question, Submission, Answer, Review
- Use cases: CreateTest, SubmitAnswers, ScoreWithAI, ReviewSubmission
- Delivery: REST API handlers, JSON serialization
- Infrastructure: PostgreSQL repos, Groq AI client, message broker, email sender
- **Status**: COMPLIANT - Clear bounded contexts identified

✅ **SOLID Principles (Principle II)**: 
- Repository interfaces in domain layer, implementations in infrastructure
- Use case layer depends on abstractions (repository interfaces)
- Each use case has single responsibility (CreateTest, AddQuestion, etc.)
- **Status**: COMPLIANT - Design enforces dependency inversion

✅ **Testability (Principle III)**: 
- All dependencies injectable via interfaces
- Domain logic testable without external services
- Contract tests for API endpoints (FR-requirement alignment)
- Target: 70% overall, 80%+ for scoring/submission critical paths
- **Status**: COMPLIANT - Test strategy defined

### Operational Gates

✅ **Observability (Principle IV)**: 
- Structured logging (slog) for all operations
- Prometheus metrics (request counts, latencies, AI scoring duration)
- OpenTelemetry tracing for request flows (especially AI async path)
- Health check endpoint for service/database/message broker
- **Status**: COMPLIANT - Full observability stack planned

✅ **Automation (Principle V)**: 
- GitHub Actions for CI/CD (lint, test, build on every PR)
- Automated security scanning (gosec, dependency checks)
- Database migrations automated (goose/migrate)
- **Status**: COMPLIANT - CI/CD pipeline required

✅ **Documentation (Principle VI)**: 
- OpenAPI 3.0 spec for all REST endpoints
- README with setup instructions
- ADRs for message broker selection, auth strategy
- Inline comments for complex domain logic
- **Status**: COMPLIANT - Documentation plan established

### Security & Compliance Gates

✅ **Security (Principle VII)**: 
- Input validation on all endpoints (email format, required fields)
- JWT for creator/reviewer authentication
- Secure token generation for participant test access
- No sensitive data in logs (passwords, tokens, PII filtered)
- **Status**: COMPLIANT - Security model defined

✅ **Secrets Management (Principle VIII)**: 
- Database credentials via ENV
- JWT signing key via ENV
- Groq API key via ENV
- Email service credentials via ENV
- **Status**: COMPLIANT - No secrets in code

✅ **Compliance & Auditability (Principle IX)**: 
- All entities have created_at, updated_at, deleted_at timestamps
- Soft delete preserves audit trail
- Submission/review actions logged with user context
- **Status**: COMPLIANT - Audit trail designed

### Performance & Scalability Gates

✅ **Performance First (Principle X)**: 
- Database indexes on frequently queried fields (test.access_token, submission.test_id)
- Async AI processing (non-blocking for participants)
- Connection pooling for database
- Performance benchmarks for critical endpoints
- **Status**: COMPLIANT - Performance-conscious design

✅ **Scalability & Fault Tolerance (Principle XI)**: 
- Stateless API design (JWT for session)
- Async worker can scale independently
- AI scoring failures don't block manual review (graceful degradation)
- Circuit breaker for Groq AI calls (prevent cascade failure)
- **Status**: COMPLIANT - Resilience patterns identified

### Data Management Gates

✅ **Database Best Practices (Principle XII)**: 
- Repository pattern isolates data access
- Goose/migrate for schema versioning
- Parameterized queries (SQL injection prevention)
- Foreign keys enforce referential integrity
- All schema changes documented in migrations
- **Status**: COMPLIANT - Database discipline enforced

### API & Integration Gates

✅ **API & Backward Compatibility (Principle XIII)**: 
- Versioned API endpoints (/api/v1/...)
- OpenAPI contract defines stable interface
- Additive changes only for minor versions
- **Status**: COMPLIANT - Versioning strategy defined

✅ **Idempotency (Principle XIV)**: 
- Submission endpoint uses unique constraint (email+test_id+version)
- Review updates are upserts (safe retries)
- Password reset tokens single-use
- **Status**: COMPLIANT - Idempotency for critical mutations

✅ **Rate Limiting (Principle XV)**: 
- Rate limit on participant submission endpoint (prevent spam)
- Rate limit on AI scoring queue (prevent Groq API exhaustion)
- Auth endpoints (login, register) rate limited (brute force protection)
- **Status**: COMPLIANT - Rate limiting planned

### Resilience & Configuration Gates

✅ **Error Handling (Principle XVI)**: 
- Consistent error response format (RFC 7807 Problem Details)
- Error codes for programmatic handling
- Internal errors logged, sanitized errors returned to client
- **Status**: COMPLIANT - Error handling standardized

✅ **Graceful Shutdown (Principle XVII)**: 
- HTTP server graceful shutdown (finish in-flight requests)
- Worker drains message queue before exit
- Database connections closed cleanly
- **Status**: COMPLIANT - Lifecycle management planned

✅ **Configuration Management (Principle XVIII)**: 
- ENV for secrets (DB, JWT, API keys)
- YAML for application config (timeouts, feature flags)
- Config validation on startup (fail-fast)
- **Status**: COMPLIANT - Externalized configuration

### Quality Gates

✅ **Minimal Dependencies (Principle XIX)**: 
- Standard library where possible (net/http, encoding/json)
- Established libraries only (gorilla/mux, testify, goose)
- No experimental or unmaintained dependencies
- **Status**: COMPLIANT - Dependency audit planned

⚠️ **I18n/L10n Awareness (Principle XX)**: 
- Baseline: English only (out of scope per spec assumptions)
- UTF-8 support throughout
- Future: design allows string externalization
- **Status**: DEFERRED - Baseline version English-only

✅ **Continuous Improvement (Principle XXI)**: 
- Retrospectives after each phase
- Performance metrics inform optimization
- Code review enforces principle adherence
- **Status**: COMPLIANT - Improvement process defined

### Gate Summary

- **Total Principles**: 21
- **Compliant**: 20
- **Deferred**: 1 (I18n - out of scope for baseline)
- **Violations**: 0

**Overall Assessment**: ✅ **PASS** - All applicable gates satisfied. Single deferral (I18n) justified by baseline scope.

## Project Structure

### Documentation (this feature)

```text
specs/001-assessment-system-baseline/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model
├── quickstart.md        # Phase 1 setup guide
├── contracts/           # Phase 1 API contracts
│   └── openapi.yaml     # OpenAPI 3.0 specification
└── tasks.md             # Phase 2 task list (generated later)
```

### Source Code (repository root)

```text
assessly-be/
├── cmd/
│   ├── api/             # HTTP API server entrypoint
│   │   └── main.go
│   └── worker/          # Async AI worker entrypoint
│       └── main.go
├── internal/
│   ├── domain/          # Clean Architecture: Domain layer
│   │   ├── user.go      # User entity + role enum
│   │   ├── test.go      # Test entity + access token
│   │   ├── question.go  # Question entity
│   │   ├── submission.go # Submission entity
│   │   ├── answer.go    # Answer entity
│   │   ├── review.go    # Review entity
│   │   └── errors.go    # Domain errors
│   ├── usecase/         # Clean Architecture: Use case layer
│   │   ├── auth/        # Authentication use cases
│   │   ├── test/        # Test management use cases
│   │   ├── submission/  # Submission use cases
│   │   ├── scoring/     # AI scoring use cases
│   │   └── review/      # Review use cases
│   ├── delivery/        # Clean Architecture: Delivery layer
│   │   ├── http/        # REST API handlers
│   │   │   ├── router.go
│   │   │   ├── middleware/ # JWT auth, logging, CORS
│   │   │   ├── auth_handler.go
│   │   │   ├── test_handler.go
│   │   │   ├── submission_handler.go
│   │   │   └── review_handler.go
│   │   └── worker/      # Message consumer for AI scoring
│   │       └── scoring_consumer.go
│   └── infrastructure/  # Clean Architecture: Infrastructure layer
│       ├── repository/  # Database implementations
│       │   ├── postgres/
│       │   └── interfaces.go  # Repository interfaces
│       ├── aiclient/    # Groq AI client
│       ├── queue/       # Message broker client (RabbitMQ/Redis/SQS)
│       ├── email/       # Email sender (password reset)
│       └── config/      # Configuration loader
├── migrations/          # Database migrations (goose)
│   ├── 001_create_users.sql
│   ├── 002_create_tests.sql
│   ├── 003_create_questions.sql
│   ├── 004_create_submissions.sql
│   ├── 005_create_answers.sql
│   └── 006_create_reviews.sql
├── tests/
│   ├── contract/        # API contract tests
│   ├── integration/     # Integration tests
│   └── unit/            # Unit tests (alongside source)
├── docs/
│   └── adr/             # Architecture Decision Records
├── deployments/
│   ├── docker-compose.yml
│   └── Dockerfile
├── .github/
│   └── workflows/
│       └── ci.yml       # GitHub Actions pipeline
├── go.mod
├── go.sum
├── Makefile            # Build, test, migrate commands
└── README.md
```

**Structure Decision**: Go standard project layout with Clean Architecture layering. API and worker are separate binaries (cmd/) sharing internal packages (internal/). Four-layer architecture (domain/usecase/delivery/infrastructure) enforced via package boundaries. Migrations in dedicated folder for Goose. Tests organized by type (unit/integration/contract) per constitution requirement.

---

## Phase 0: Research & Technical Decisions

**Status**: ✅ **COMPLETED**  
**Deliverable**: [research.md](research.md)

### Research Summary

All technical unknowns from Technical Context have been resolved. Seven key technical decisions documented with rationale:

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **TD-001: Message Broker** | Redis Streams | Simplicity, in-memory performance, persistence support, consumer groups for scaling. RabbitMQ over-engineered for baseline single-queue use case. |
| **TD-002: Email Service** | SMTP with pluggable backend (gomail) | Flexibility, no vendor lock-in. Can use Gmail, SendGrid, Mailgun, SES, or MailHog for local dev. |
| **TD-003: HTTP Router** | chi | Lightweight, idiomatic Go, context-based, active maintenance. Gorilla/mux archived, gin/fiber too opinionated or non-standard. |
| **TD-004: Migration Tool** | golang-migrate/migrate | Go-native, database-agnostic, library+CLI flexibility, production-ready. Preferred over goose for pure SQL migrations. |
| **TD-005: JWT Library** | golang-jwt/jwt (v5) | Industry standard, secure, simple API. Preferred over lestrrat-go/jwx (over-featured) and cristalhq/jwt (smaller community). |
| **TD-006: Groq Integration** | Custom net/http wrapper | No official SDK, simple REST API. Custom wrapper provides retry logic, circuit breaker, timeout control, and testability. |
| **TD-007: Testing Strategy** | Layered (unit/integration/contract) | Unit tests (80%+ for domain/usecase), integration tests (testcontainers), contract tests (OpenAPI validation). |

### Key Architecture Decisions

**AIScorer Interface** (Strategy Pattern):
```go
type AIScorer interface {
    ScoreAnswer(ctx context.Context, question, expected, answer string) (*ScoringResult, error)
}
```
Allows swapping Groq with alternative AI providers (Claude, OpenAI, local models) without changing use case layer.

**Async Scoring Flow**:
1. Participant submits → Submission created (HTTP response 201)
2. Worker picks submission from Redis Stream → Calls Groq API
3. Worker stores Review with ai_score, ai_feedback
4. Participant polls GET /submissions/{id} for results

**Email Access Token Flow**:
- Secure JWT token generated with (test_id, email, expiry=1h)
- Token sent via email link: `https://assessly.example.com/take-test?token=...`
- Token validated on submission POST (cannot reuse if retakes disabled)

### Research Artifacts

- **research.md**: Full decision rationale with alternatives matrix
- **ADRs** (planned): Message broker selection, auth strategy, AI integration

---

## Phase 1: Design & Contracts

**Status**: ✅ **COMPLETED**  
**Deliverables**: 
- [data-model.md](data-model.md)
- [contracts/openapi.yaml](contracts/openapi.yaml)
- [quickstart.md](quickstart.md)
- Agent context updated (`.github/agents/copilot-instructions.md`)

### Data Model Summary

**Database**: PostgreSQL 14+ with 6 core tables

**Entities**:

| Entity | Purpose | Key Fields | Relationships |
|--------|---------|------------|---------------|
| **users** | Creators/reviewers (auth) | id, email, password_hash, role | 1:N tests, 1:N reviews |
| **tests** | Test definitions | id, creator_id, title, allow_retakes, is_published | 1:N questions, 1:N submissions |
| **questions** | Essay prompts with rubrics | id, test_id, text, expected_answer, order_num | N:1 test, 1:N answers |
| **submissions** | Participant test completions | id, test_id, access_email, ai_total_score, manual_total_score | N:1 test, 1:N answers |
| **answers** | Individual essay responses | id, submission_id, question_id, text | N:1 submission, N:1 question, 1:1 review |
| **reviews** | AI + manual scoring | id, answer_id, ai_score, ai_feedback, manual_score, manual_feedback, reviewer_id | 1:1 answer, N:1 reviewer |

**Key Design Decisions**:
- **Per-answer scoring**: Each answer has ONE review with separate ai_score and manual_score fields (clarification #5)
- **Anonymous participants**: access_email in submissions table, NO foreign key to users (clarification context)
- **Denormalized totals**: ai_total_score and manual_total_score in submissions for performance
- **Retake enforcement**: Application-level check (count submissions for test_id + access_email)
- **Soft delete**: Planned for future (deleted_at column), out of scope for baseline

**Migrations Strategy**:
- 6 up/down migration pairs (001-006)
- Sequential numbering (golang-migrate convention)
- SQL-only (no Go code in migrations for reproducibility)
- UUID primary keys via uuid-ossp extension

**ERD**: Full entity-relationship diagram in data-model.md with cascade rules and indexes.

### API Contract Summary

**OpenAPI 3.0 Specification**: [contracts/openapi.yaml](contracts/openapi.yaml)

**Endpoint Groups**:

| Group | Endpoints | Purpose |
|-------|-----------|---------|
| **Auth** | POST /auth/register, POST /auth/login, POST /auth/reset-password, PUT /auth/reset-password/{token} | User authentication, JWT issuance, password reset flow |
| **Tests** | GET/POST /tests, GET/PUT/DELETE /tests/{id}, POST /tests/{id}/publish, POST /tests/{id}/unpublish | Test CRUD and lifecycle management |
| **Questions** | GET/POST /tests/{id}/questions, GET/PUT/DELETE /tests/{id}/questions/{qid} | Question management within tests |
| **Submissions** | POST /submissions/access, POST /submissions, GET /submissions/{id}, GET /tests/{id}/submissions | Participant test access, submission, result retrieval |
| **Reviews** | GET/PUT /reviews/{answerId} | Manual review addition by reviewers |

**Authentication**:
- **BearerAuth**: JWT tokens for creators/reviewers (Authorization: Bearer <token>)
- **AccessToken**: Participant tokens from email (X-Access-Token header or query param)

**Key Endpoints**:

**Creator Flow**:
1. POST /auth/register → Create creator account
2. POST /auth/login → Get JWT token
3. POST /tests → Create test (draft)
4. POST /tests/{id}/questions → Add questions (min 1)
5. POST /tests/{id}/publish → Publish test

**Participant Flow**:
1. POST /submissions/access → Request access token via email
2. POST /submissions → Submit answers with access_token
3. GET /submissions/{id}?access_token=... → View AI scores (async, poll for results)

**Reviewer Flow**:
1. POST /auth/login → Get reviewer JWT
2. GET /tests → List all published tests
3. GET /tests/{id}/submissions → View submissions for test
4. PUT /reviews/{answerId} → Add manual score/feedback

**Response Formats**:
- Success: Domain entity JSON (User, Test, Submission, Review)
- Error: RFC 7807 Problem Details (error, code, details fields)
- Lists: Paginated with pagination metadata (page, page_size, total_items, total_pages)

**Validation Rules**:
- Email: RFC 5322 format, max 255 chars
- Password: Min 8 chars (bcrypt hashed)
- Scores: 0-100 range (enforced at DB level)
- Text fields: Max lengths documented in schema

**Contract Testing**: OpenAPI spec is the source of truth. Contract tests (tests/contract/) will validate all API responses match schema.

### Quickstart Guide Summary

**Deliverable**: [quickstart.md](quickstart.md)

**Setup Options**:
1. **Docker Compose** (recommended): One-command setup with PostgreSQL, Redis, API, worker
2. **Manual setup**: Local PostgreSQL, Redis, Go installation with detailed instructions

**Time to First Request**: ~10 minutes (Docker Compose path)

**Key Sections**:
- Prerequisites (Go 1.21+, PostgreSQL 14+, Redis 6+)
- Quick setup (docker-compose up -d)
- Manual setup (database creation, migration commands)
- Configuration (environment variables with defaults)
- Development workflow (testing, migrations, code quality)
- Example API flows (creator → participant → reviewer)
- Troubleshooting (database, Redis, email, AI scoring)
- Useful commands cheat sheet

**Development Tools Recommended**:
- MailHog/MailCatcher (local email testing)
- Delve (Go debugger)
- golangci-lint (code linting)
- gosec (security scanning)

### Agent Context Update

**Status**: ✅ **COMPLETED**

Agent context file updated with:
- Language: Go (Golang) >= 1.21
- Database: PostgreSQL 14+ (transactions, JSONB, full-text search)
- Project type: RESTful web service with async workers

Agent now has full context for code generation tasks.

---

## Phase 2 Preparation

**Status**: ⏸️ **PENDING** (to be completed in tasks generation phase)

Phase 2 will involve:
1. Generating tasks.md with dependency-ordered implementation tasks
2. Breaking work into PRs (implement → test → document cycle)
3. Defining CI/CD pipeline in GitHub Actions

**Recommended Task Groups** (preliminary):
1. **Bootstrap** (migrations, config, project structure)
2. **Domain layer** (entities, repository interfaces)
3. **Infrastructure layer** (Postgres repos, Redis queue, Groq client, SMTP)
4. **Use case layer** (auth, test management, submission, scoring, review)
5. **Delivery layer** (HTTP handlers, middleware, worker)
6. **Testing** (unit tests, integration tests, contract tests)
7. **Deployment** (Dockerfile, docker-compose, GitHub Actions)

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| **Groq API rate limits** | Medium | High (AI scoring fails) | Circuit breaker pattern, exponential backoff, fallback to manual review only |
| **Redis queue failures** | Low | High (AI scoring blocked) | Health checks, automatic worker restart, queue monitoring alerts |
| **Database migrations conflict** | Medium | Medium (deployment errors) | Migration testing in CI, versioned schema, rollback procedures |
| **JWT secret compromise** | Low | Critical (auth bypass) | Secret rotation procedure, short expiry, secure storage |
| **AI scoring accuracy concerns** | Medium | Medium (user trust) | Manual review always available, score comparison metrics, feedback loop to improve prompts |
| **Email deliverability issues** | Medium | Medium (participants can't access tests) | Email server monitoring, fallback to alternative SMTP, retry logic |

---

## Success Metrics (Post-Implementation)

| Metric | Target | Validation Method |
|--------|--------|-------------------|
| **Test Coverage** | 70% overall, 80%+ critical paths | `go test -cover ./...` in CI |
| **API Response Time** | < 2s for 95% of non-AI requests | Prometheus metrics, load testing |
| **AI Scoring Latency** | < 30s for 90% of submissions | Worker metrics, Redis queue depth |
| **Concurrent Submissions** | 100+ without degradation | Load testing with k6/vegeta |
| **Zero Security Vulnerabilities** | No high/critical findings | gosec, Snyk scans in CI |
| **API Contract Compliance** | 100% endpoints match OpenAPI spec | Contract tests pass |
| **Deployment Success** | One-command Docker Compose setup | Quickstart validation |

---

## Next Steps

### Immediate (Ready to Execute)

1. **Generate tasks.md**: Run speckit.tasks workflow to break Phase 2 into actionable tasks
2. **Create feature branch**: `git checkout -b 001-assessment-system-baseline`
3. **Initialize project**: Run `go mod init github.com/<org>/assessly-be`
4. **Setup CI/CD**: Create `.github/workflows/ci.yml` per constitution

### Phase 2 Implementation Order

1. **Bootstrap** (Day 1-2):
   - Project structure (cmd/, internal/, migrations/)
   - Configuration loader (ENV + YAML)
   - Database migrations (001-006)
   - Makefile (build, test, migrate, run commands)

2. **Domain + Infrastructure** (Day 3-5):
   - Domain entities and interfaces
   - PostgreSQL repositories
   - Redis queue client
   - Groq AI client
   - SMTP email sender

3. **Use Cases** (Day 6-9):
   - Auth use cases (register, login, password reset)
   - Test management use cases
   - Submission use cases
   - Scoring use cases
   - Review use cases

4. **Delivery** (Day 10-12):
   - HTTP handlers
   - Middleware (auth, logging, CORS)
   - Worker consumer
   - OpenAPI contract tests

5. **Testing & Deployment** (Day 13-15):
   - Unit tests (80%+ coverage)
   - Integration tests (testcontainers)
   - Dockerfile + docker-compose
   - GitHub Actions CI/CD
   - Documentation finalization

**Estimated Timeline**: 15 working days (3 weeks at 60% utilization)

---

## Appendices

### A. Referenced Documents

- [Feature Specification](spec.md): All functional requirements (FR-001 to FR-038)
- [Research Document](research.md): Technical decision rationale (TD-001 to TD-007)
- [Data Model](data-model.md): Database schema, ERD, migrations
- [API Contracts](contracts/openapi.yaml): OpenAPI 3.0 specification
- [Quickstart Guide](quickstart.md): Development setup instructions
- [Constitution](.specify/memory/constitution.md): 21 architectural principles

### B. Glossary

- **Clean Architecture**: Architectural pattern with domain/usecase/delivery/infrastructure layers
- **Creator**: User role that creates and manages tests
- **Reviewer**: User role that adds manual scores to submissions
- **Participant**: Anonymous user taking a test (no account required)
- **Submission**: Complete set of answers for a test by one participant
- **Review**: AI + manual scoring for a single answer
- **Access Token**: JWT token sent via email for participant test access
- **Retake**: Submitting a test multiple times (configurable per test)
- **AI Scoring**: Automatic grading using Groq language models

### C. Open Questions (None)

All clarifications resolved in specification phase. No outstanding questions.

---

**Plan Status**: ✅ **COMPLETE** - Ready for task generation (speckit.tasks workflow)  
**Phase 0**: ✅ Research complete  
**Phase 1**: ✅ Design artifacts complete  
**Phase 2**: ⏸️ Awaiting task generation

**Last Updated**: 2026-03-26  
**Next Action**: Run `speckit.tasks` to generate implementation task list

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
