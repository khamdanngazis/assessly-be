# Research: Flexible Assessment System Baseline

**Feature**: 001-assessment-system-baseline  
**Phase**: 0 (Outline & Research)  
**Date**: 2026-03-26

## Purpose

This document consolidates research findings for technical decisions required before detailed design. Each decision is documented with rationale, alternatives considered, and final recommendation.

## Technical Decisions

### TD-001: Message Broker Selection

**Decision**: Use **Redis Streams** for async AI scoring queue

**Rationale**:
- **Simplicity**: Redis already commonly used for caching; reduces infrastructure complexity
- **Performance**: In-memory operations, low latency for queue operations
- **Persistence**: AOF/RDB persistence options for message durability
- **Consumer Groups**: Built-in consumer group support for worker scaling
- **Monitoring**: Well-established monitoring tools (Redis Exporter for Prometheus)
- **Development**: Simpler local development setup than RabbitMQ

**Alternatives Considered**:

| Option | Pros | Cons | Why Not Chosen |
|--------|------|------|----------------|
| **RabbitMQ** | - Enterprise-grade messaging<br>- Advanced routing<br>- Strong durability guarantees<br>- AMQP standard | - Additional infrastructure (separate service)<br>- More complex setup<br>- Overkill for baseline (single queue) | Over-engineered for baseline scope; added operational complexity |
| **AWS SQS** | - Fully managed<br>- Infinite scalability<br>- No infrastructure management | - Cloud vendor lock-in<br>- Additional cost<br>- Local development requires LocalStack | Vendor lock-in and cost not justified for baseline; complicates local dev |
| **PostgreSQL LISTEN/NOTIFY** | - No additional infrastructure<br>- Transactional guarantees | - Not designed for queuing<br>- No built-in retry/DLQ<br>- Connection management complexity | Missing queue semantics; brittle for production workloads |

**Implementation Notes**:
- Use Redis Streams (XADD/XREADGROUP) for queue operations
- Consumer groups for multiple worker instances
- Configurable retry count and dead letter queue (separate stream)
- Connection pooling via go-redis/redis library

---

### TD-002: Email Service Provider

**Decision**: Use **SMTP with pluggable backend** (standard library net/smtp + gomail)

**Rationale**:
- **Flexibility**: Can use any SMTP-compatible service (Gmail, SendGrid, Mailgun, SES)
- **Low Vendor Lock-in**: SMTP is standard protocol
- **Cost Control**: Choose provider based on volume/budget
- **Local Development**: Can use MailHog/MailCatcher for testing
- **Simple Integration**: Go standard library + lightweight wrapper (gomail)

**Alternatives Considered**:

| Option | Pros | Cons | Why Not Chosen |
|--------|------|------|----------------|
| **SendGrid API** | - Modern REST API<br>- Rich features (templates, analytics)<br>- Reliable delivery | - Vendor lock-in<br>- API-specific code<br>- Paid service required | Direct API lock-in not justified; SMTP provides flexibility |
| **AWS SES** | - Cost-effective at scale<br>- High deliverability<br>- Integrates with AWS ecosystem | - AWS lock-in<br>- Requires AWS account<br>- Complicated local dev | Cloud lock-in; baseline should be deployment-agnostic |
| **Mailgun API** | - Developer-friendly<br>- Good deliverability<br>- Analytics | - Vendor lock-in<br>- Pricing complexity | Same as SendGrid - API lock-in issue |

**Implementation Notes**:
- Use gomail library (wraps net/smtp with convenience)
- Configuration via ENV: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS
- Template password reset emails in code (baseline) or HTML files
- Async email sending (non-blocking for user requests)
- Log email send attempts for debugging (without sensitive content)

---

### TD-003: HTTP Router

**Decision**: Use **chi** router

**Rationale**:
- **Lightweight**: Minimal abstraction over net/http
- **Middleware Support**: Built-in middleware chaining (idiomatic Go)
- **Context-Based**: Leverages Go context for request-scoped values
- **Active Maintenance**: Well-maintained, modern codebase
- **Performance**: Benchmarks show excellent performance
- **Standard Library Compatible**: Uses standard http.Handler interface

**Alternatives Considered**:

| Option | Pros | Cons | Why Not Chosen |
|--------|------|------|----------------|
| **gorilla/mux** | - Most popular historically<br>- Well-documented<br>- Mature | - Archived (maintenance mode)<br>- Heavier than chi<br>- Legacy API patterns | Officially archived; community moving to alternatives |
| **gin** | - Very fast (httprouter-based)<br>- Large ecosystem<br>- Popular in Asia/China | - Custom context (not standard)<br>- Magic abstractions<br>- Less idiomatic Go | Non-standard patterns; deviates from Go idioms |
| **fiber** | - Express-like API<br>- Very fast (fasthttp-based) | - Non-standard (fasthttp not net/http)<br>- Breaks Go ecosystem compatibility<br>- Overkill for baseline | Incompatible with standard middleware; unnecessary complexity |
| **Standard library** | - No dependencies<br>- Maximum control | - Verbose<br>- No middleware chain<br>- Manual route management | Too low-level; reinventing patterns already solved by chi |

**Implementation Notes**:
- Use chi.Router with middleware stack
- Middleware: logging, recovery, CORS, JWT authentication
- Route grouping for versioning (/api/v1/...)
- Use chi.URLParam for path parameters

---

### TD-004: Database Migration Tool

**Decision**: Use **golang-migrate/migrate**

**Rationale**:
- **Go-Native**: Pure Go implementation, no external CLI dependencies
- **Database Agnostic**: Supports PostgreSQL, MySQL, SQLite, etc.
- **Flexible**: Can run as library or CLI
- **Version Control**: Up/down migrations with version tracking
- **Production Ready**: Used by many production Go applications
- **Active Maintenance**: Regular updates and bug fixes

**Alternatives Considered**:

| Option | Pros | Cons | Why Not Chosen |
|--------|------|------|----------------|
| **pressly/goose** | - Supports Go migrations (not just SQL)<br>- Embedded SQL<br>- Simple CLI | - Less database support<br>- Smaller community | Go migrations are double-edged; prefer pure SQL for clarity |
| **rubenv/sql-migrate** | - Embedded migrations<br>- Supports gorp ORM | - Less active maintenance<br>- ORM coupling | Maintenance concerns; we're not using gorp |
| **Manual SQL scripts** | - Full control<br>- No dependencies | - No version tracking<br>- Error-prone<br>- Manual rollback logic | No automation; violates Principle V (Automation) |

**Implementation Notes**:
- Migrations in /migrations directory (sequential numbering)
- Use golang-migrate CLI for local development
- Programmatic migration runner in cmd/api/main.go (auto-migrate on startup)
- SQL-only migrations (no Go code in migrations for reproducibility)
- Each migration has up/down pair (.up.sql, .down.sql)

---

### TD-005: JWT Library

**Decision**: Use **golang-jwt/jwt** (v5)

**Rationale**:
- **Standard**: Most widely used JWT library in Go ecosystem
- **Secure**: Actively maintained, security patches
- **Flexible**: Supports all common signing algorithms (HS256, RS256, etc.)
- **Simple API**: Easy to generate and validate tokens
- **Well-Documented**: Extensive examples and documentation

**Alternatives Considered**:

| Option | Pros | Cons | Why Not Chosen |
|--------|------|------|----------------|
| **lestrrat-go/jwx** | - Modern API<br>- Full JOSE support<br>- JWK handling | - More complex<br>- Heavier than needed for baseline | Over-featured for baseline; JOSE not required |
| **cristalhq/jwt** | - Minimal<br>- Fast<br>- Zero dependencies | - Less battle-tested<br>- Smaller community | Prefer established library for security-critical component |
| **Manual implementation** | - Full control<br>- Learning opportunity | - Security risks<br>- Reinventing wheel | Never roll your own crypto (Principle VII) |

**Implementation Notes**:
- Use HS256 for signing (symmetric key from ENV)
- Set expiration (24 hours for access tokens, 1 hour for reset tokens)
- Include user ID and role in claims
- Validate signature, expiration, and issuer on every request

---

### TD-006: Groq AI Integration

**Decision**: Use **standard library net/http** with custom client wrapper

**Rationale**:
- **No Official SDK**: Groq doesn't provide official Go SDK
- **Simple REST API**: Groq API is straightforward HTTP POST
- **Full Control**: Custom wrapper allows retry logic, circuit breaker, timeouts
- **Testability**: Easy to mock for tests (interface-based)
- **No Lock-In**: Not coupled to third-party SDK lifecycles

**Implementation Pattern**:

```go
type AIScorer interface {
    ScoreAnswer(ctx context.Context, question, expected, answer string) (*ScoringResult, error)
}

type GroqClient struct {
    httpClient *http.Client
    apiKey     string
    baseURL    string
}

func (g *GroqClient) ScoreAnswer(ctx context.Context, question, expected, answer string) (*ScoringResult, error) {
    // Build prompt: "Grade this essay answer: [question] Expected: [expected] Student: [answer]"
    // POST to Groq API with retry logic and circuit breaker
    // Parse response for score (0-100) and feedback text
    // Handle errors gracefully (network, rate limit, invalid response)
}
```

**Implementation Notes**:
- Timeout of 30 seconds per AI request (matches SC-004 success criteria)
- Exponential backoff for retries (3 attempts max)
- Circuit breaker pattern (after N failures, skip AI temporarily)
- Structured logging for all AI interactions (request ID, duration, errors)
- Mock implementation for tests

---

### TD-007: Testing Strategy

**Decision**: Layered testing approach (unit/integration/contract)

**Rationale**:
- **Constitution Requirement**: Principle III (Testability) mandates comprehensive testing
- **Coverage Target**: 70% minimum (constitution), 80%+ for critical paths
- **Confidence**: Multiple test layers catch different bug categories

**Test Layers**:

| Layer | Purpose | Tools | Coverage Target |
|-------|---------|-------|-----------------|
| **Unit Tests** | Test individual functions/methods in isolation | testing + testify/assert | 80%+ for domain/usecase layers |
| **Integration Tests** | Test component interactions (API + DB + queue) | testing + testcontainers | Critical flows (submission, scoring) |
| **Contract Tests** | Validate API adheres to OpenAPI spec | testing + OpenAPI validator | All endpoints |

**Test Organization**:
```
tests/
├── unit/           # Unit tests (also co-located with source *_test.go)
├── integration/    # Integration tests with real dependencies
│   ├── submission_test.go
│   ├── scoring_test.go
│   └── review_test.go
└── contract/       # API contract tests
    └── openapi_test.go
```

**Implementation Notes**:
- Use testify/assert for assertions (reduce boilerplate)
- Use testcontainers for integration tests (PostgreSQL, Redis)
- Table-driven tests for unit tests (Go idiomatic)
- Parallel test execution where possible (+testing.T.Parallel())
- Test fixtures in testdata/ directory
- Mock interfaces for external dependencies (AIScorer, EmailSender)

---

## Decision Summary

| ID | Decision | Choice | Phase Impact |
|----|----------|--------|--------------|
| TD-001 | Message Broker | Redis Streams | Data model, infrastructure, deployment |
| TD-002 | Email Service | SMTP (gomail) | Infrastructure, configuration |
| TD-003 | HTTP Router | chi | Delivery layer, middleware |
| TD-004 | Migration Tool | golang-migrate | Database setup, CI/CD |
| TD-005 | JWT Library | golang-jwt/jwt | Authentication, middleware |
| TD-006 | AI Integration | Custom net/http wrapper | Infrastructure, scoring usecase |
| TD-007 | Testing | Unit/Integration/Contract | All layers, CI/CD |

## Next Steps (Phase 1)

With technical decisions finalized, proceed to Phase 1 deliverables:

1. **data-model.md**: Define database schema with entity relationships
2. **contracts/openapi.yaml**: Specify REST API endpoints with OpenAPI 3.0
3. **quickstart.md**: Local development setup guide
4. **Update agent context**: Add technology choices to agent knowledge base

All unknowns resolved. No NEEDS CLARIFICATION markers remain.
