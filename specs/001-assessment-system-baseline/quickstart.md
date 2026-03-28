# Quickstart Guide: Assessly Backend

**Feature**: 001-assessment-system-baseline  
**Last Updated**: 2026-03-26

## Overview

This guide walks you through setting up the Assessly backend for local development, running tests, and making your first API request.

**Time to first request**: ~10 minutes

**Prerequisites**:
- Go 1.21 or higher
- PostgreSQL 14 or higher
- Redis 6 or higher (for AI scoring queue)
- Git
- (Optional) Docker and Docker Compose for containerized setup

---

## Quick Setup (Docker Compose)

The fastest way to get started is with Docker Compose:

### 1. Clone and Setup

```bash
# Clone repository
git clone <repository-url> assessly-be
cd assessly-be

#Check out baseline feature branch
git checkout 001-assessment-system-baseline

# Copy environment template
cp .env.example .env

# Edit .env with your settings (see Configuration section)
nano .env
```

### 2. Start Services

```bash
# Start PostgreSQL, Redis, and API server
docker-compose up -d

# Check logs
docker-compose logs -f api

# Wait for "Server listening on :8080" message
```

### 3. Verify Setup

```bash
# Health check
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","database":"connected","redis":"connected"}
```

### 4. Create First User

```bash
# Register a creator account
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "creator@example.com",
    "password": "SecurePass123!",
    "role": "creator"
  }'

# Expected response:
# {"id":"<uuid>","email":"creator@example.com","role":"creator","created_at":"2026-03-26T..."}
```

### 5. Login and Get Token

```bash
# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "creator@example.com",
    "password": "SecurePass123!"
  }'

# Expected response:
# {"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...","user":{...}}

# Save token for subsequent requests:
export TOKEN="<token-from-response>"
```

### 6. Create Your First Test

```bash
# Create test
curl -X POST http://localhost:8080/api/v1/tests \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "title": "Programming Concepts Test",
    "description": "Test your understanding of key programming principles",
    "allow_retakes": false
  }'

# Save test ID from response:
export TEST_ID="<id-from-response>"

# Add question
curl -X POST http://localhost:8080/api/v1/tests/$TEST_ID/questions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "text": "What are the key principles of Clean Architecture?",
    "expected_answer": "Clean Architecture emphasizes separation of concerns, with business logic independent of frameworks, UI, and databases. Key principles include dependency inversion, testability, and clear boundaries between layers.",
    "order_num": 1
  }'

# Publish test
curl -X POST http://localhost:8080/api/v1/tests/$TEST_ID/publish \
  -H "Authorization: Bearer $TOKEN"
```

**✅ You're ready!** Your API is running with a published test. Participants can now request access and submit answers.

---

## Manual Setup (Without Docker)

If you prefer to run services locally without containers:

### 1. Install Dependencies

**PostgreSQL**:
```bash
# Ubuntu/Debian
sudo apt-get install postgresql-14

# macOS
brew install postgresql@14

# Start PostgreSQL
sudo systemctl start postgresql  # Linux
brew services start postgresql@14  # macOS
```

**Redis**:
```bash
# Ubuntu/Debian
sudo apt-get install redis-server

# macOS
brew install redis

# Start Redis
sudo systemctl start redis  # Linux
brew services start redis  # macOS
```

**Go**:
```bash
# Download from https://go.dev/dl/
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH=$PATH:/usr/local/go/bin
```

### 2. Setup Database

```bash
# Create database user
sudo -u postgres psql -c "CREATE USER assessly WITH PASSWORD 'assessly_dev';"

# Create database
sudo -u postgres psql -c "CREATE DATABASE assessly_dev OWNER assessly;"

# Verify connection
psql -h localhost -U assessly -d assessly_dev -c "SELECT version();"
```

### 3. Clone and Configure

```bash
# Clone repository
git clone <repository-url> assessly-be
cd assessly-be

# Checkout feature branch
git checkout 001-assessment-system-baseline

# Install Go dependencies
go mod download

# Copy environment template
cp .env.example .env

# Edit configuration
nano .env
```

**Minimal .env configuration**:
```env
# Server
PORT=8080
ENV=development

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=assessly
DB_PASSWORD=assessly_dev
DB_NAME=assessly_dev
DB_SSL_MODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_SECRET=change-this-in-production-use-random-64-chars
JWT_EXPIRY_HOURS=24

# Groq AI
GROQ_API_KEY=your-groq-api-key-here
GROQ_MODEL=llama-3-70b-8192

# SMTP (for password reset emails)
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USER=
SMTP_PASS=
SMTP_FROM=noreply@assessly.local

# Email testing tool (MailCatcher/MailHog)
# Install: gem install mailcatcher or brew install mailhog
# Run: mailcatcher or mailhog
# Web UI: http://localhost:1080
```

### 4. Run Migrations

```bash
# Install golang-migrate CLI (if not already)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path ./migrations -database "postgres://assessly:assessly_dev@localhost:5432/assessly_dev?sslmode=disable" up

# Verify tables created
psql -h localhost -U assessly -d assessly_dev -c "\dt"
```

### 5. Start Services

**Terminal 1 - API Server**:
```bash
# Run server
go run cmd/api/main.go

# Or build and run
go build -o bin/api cmd/api/main.go
./bin/api

# Wait for "Server listening on :8080"
```

**Terminal 2 - Worker (AI Scoring)**:
```bash
# Run worker
go run cmd/worker/main.go

# Or build and run
go build -o bin/worker cmd/worker/main.go
./bin/worker

# Wait for "Worker started, listening to queue: ai-scoring"
```

**Terminal 3 - Email Testing**:
```bash
# Option 1: MailCatcher (Ruby)
mailcatcher
# Web UI: http://localhost:1080

# Option 2: MailHog (Go)
mailhog
# Web UI: http://localhost:8025
```

### 6. Verify Setup

```bash
# Health check
curl http://localhost:8080/health

# Create user and follow steps 4-6 from Docker Compose section above
```

---

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | API server port | `8080` | No |
| `ENV` | Environment (development/production) | `development` | No |
| `DB_HOST` | PostgreSQL host | `localhost` | Yes |
| `DB_PORT` | PostgreSQL port | `5432` | No |
| `DB_USER` | PostgreSQL username | - | Yes |
| `DB_PASSWORD` | PostgreSQL password | - | Yes |
| `DB_NAME` | PostgreSQL database name | - | Yes |
| `DB_SSL_MODE` | SSL mode (disable/require) | `disable` | No |
| `REDIS_HOST` | Redis host | `localhost` | Yes |
| `REDIS_PORT` | Redis port | `6379` | No |
| `REDIS_PASSWORD` | Redis password (if auth enabled) | - | No |
| `REDIS_DB` | Redis database number | `0` | No |
| `JWT_SECRET` | Secret key for JWT signing | - | Yes |
| `JWT_EXPIRY_HOURS` | JWT token expiry | `24` | No |
| `GROQ_API_KEY` | Groq API key for AI scoring | - | Yes |
| `GROQ_MODEL` | Groq model to use | `llama-3-70b-8192` | No |
| `SMTP_HOST` | SMTP server host | - | Yes |
| `SMTP_PORT` | SMTP server port | `587` | No |
| `SMTP_USER` | SMTP username | - | No |
| `SMTP_PASS` | SMTP password | - | No |
| `SMTP_FROM` | From email address | - | Yes |

**Getting Groq API Key**:
1. Sign up at https://console.groq.com/
2. Navigate to API Keys section
3. Create new key and copy to `GROQ_API_KEY`

---

## Project Structure

```
assessly-be/
├── cmd/
│   ├── api/              # API server entrypoint
│   │   └── main.go       # Server initialization
│   └── worker/           # Worker entrypoint
│       └── main.go       # Worker initialization
├── internal/
│   ├── domain/           # Business entities and interfaces
│   │   ├── user.go
│   │   ├── test.go
│   │   ├── question.go
│   │   ├── submission.go
│   │   ├── answer.go
│   │   └── review.go
│   ├── usecase/          # Business logic
│   │   ├── auth/
│   │   ├── test/
│   │   ├── submission/
│   │   └── review/
│   ├── delivery/         # HTTP handlers (API layer)
│   │   ├── http/
│   │   │   ├── auth_handler.go
│   │   │   ├── test_handler.go
│   │   │   ├── submission_handler.go
│   │   │   └── review_handler.go
│   │   └── middleware/   # HTTP middleware (auth, logging, CORS)
│   └── infrastructure/   # External dependencies
│       ├── postgres/     # PostgreSQL repositories
│       ├── redis/        # Redis queue client
│       ├── groq/         # Groq AI client
│       └── smtp/         # Email sender
├── migrations/           # Database migrations
│   ├── 001_create_users.up.sql
│   ├── 001_create_users.down.sql
│   ├── ...
├── tests/
│   ├── unit/            # Unit tests
│   ├── integration/     # Integration tests
│   └── contract/        # API contract tests
├── .env.example         # Environment template
├── .env                 # Local environment (git-ignored)
├── docker-compose.yml   # Docker services definition
├── Dockerfile           # API container image
├── go.mod               # Go dependencies
├── go.sum               # Dependency checksums
└── README.md            # Project documentation
```

---

## Development Workflow

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run only unit tests
go test ./tests/unit/...

# Run integration tests (requires running DB and Redis)
go test ./tests/integration/...

# Run specific test
go test -run TestUserRegistration ./internal/usecase/auth/...
```

### Database Migrations

```bash
# Create new migration
migrate create -ext sql -dir migrations -seq add_user_status

# Run migrations up
migrate -path ./migrations -database "postgres://...connection-string..." up

# Rollback last migration
migrate -path ./migrations -database "postgres://...connection-string..." down 1

# Check migration version
migrate -path ./migrations -database "postgres://...connection-string..." version

# Force migration version (if stuck)
migrate -path ./migrations -database "postgres://...connection-string..." force <version>
```

### Code Quality

```bash
# Format code
go fmt ./...

# Lint code (install golangci-lint first)
golangci-lint run

# Vet code
go vet ./...

# Security scan (install gosec first)
gosec ./...
```

### Debugging

**Enable debug logging**:
```env
# In .env
LOG_LEVEL=debug
```

**Use Delve debugger**:
```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug API
dlv debug cmd/api/main.go

# Debug tests
dlv test ./internal/usecase/auth/...
```

**Database query logging**:
```go
// In internal/infrastructure/postgres/db.go
// Set log level to DEBUG for pgx
config.LogLevel = pgx.LogLevelDebug
```

---

## Testing the API

### Example Flow: Creator → Participant → Reviewer

**1. Creator creates test**:
```bash
# Register creator
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"creator@test.com","password":"Pass123!","role":"creator"}'

# Login
TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"creator@test.com","password":"Pass123!"}' \
  | jq -r '.token')

# Create test
TEST_ID=$(curl -X POST http://localhost:8080/api/v1/tests \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test 1","allow_retakes":false}' \
  | jq -r '.id')

# Add question
curl -X POST http://localhost:8080/api/v1/tests/$TEST_ID/questions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "text":"Explain Clean Architecture",
    "expected_answer":"Separation of concerns with independent business logic",
    "order_num":1
  }'

# Publish
curl -X POST http://localhost:8080/api/v1/tests/$TEST_ID/publish \
  -H "Authorization: Bearer $TOKEN"
```

**2. Participant takes test**:
```bash
# Request access
curl -X POST http://localhost:8080/api/v1/submissions/access \
  -H "Content-Type: application/json" \
  -d '{"test_id":"'$TEST_ID'","email":"student@test.com"}'

# Check email (MailHog/MailCatcher UI) and copy access token
ACCESS_TOKEN="<token-from-email>"

# Submit answers
SUBMISSION_ID=$(curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Content-Type: application/json" \
  -d '{
    "access_token":"'$ACCESS_TOKEN'",
    "answers":[{
      "question_id":"<question-id>",
      "text":"Clean Architecture separates business logic from frameworks..."
    }]
  }' | jq -r '.id')

# Wait ~30 seconds for AI scoring

# View results
curl "http://localhost:8080/api/v1/submissions/$SUBMISSION_ID?access_token=$ACCESS_TOKEN"
```

**3. Reviewer adds manual score**:
```bash
# Register reviewer
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"reviewer@test.com","password":"Pass123!","role":"reviewer"}'

# Login
REVIEWER_TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"reviewer@test.com","password":"Pass123!"}' \
  | jq -r '.token')

# View submission
curl http://localhost:8080/api/v1/submissions/$SUBMISSION_ID \
  -H "Authorization: Bearer $REVIEWER_TOKEN"

# Add manual review (get answer_id from submission response)
ANSWER_ID="<answer-id-from-submission>"
curl -X PUT http://localhost:8080/api/v1/reviews/$ANSWER_ID \
  -H "Authorization: Bearer $REVIEWER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"manual_score":95.0,"manual_feedback":"Excellent work!"}'
```

---

## Troubleshooting

### Database Connection Issues

**Error**: `pq: password authentication failed`
```bash
# Check PostgreSQL is running
sudo systemctl status postgresql

# Reset password
sudo -u postgres psql -c "ALTER USER assessly WITH PASSWORD 'new_password';"

# Update .env with new password
```

**Error**: `database "assessly_dev" does not exist`
```bash
# Create database
sudo -u postgres psql -c "CREATE DATABASE assessly_dev OWNER assessly;"
```

### Redis Connection Issues

**Error**: `dial tcp 127.0.0.1:6379: connect: connection refused`
```bash
# Check Redis is running
sudo systemctl status redis

# Start Redis
sudo systemctl start redis

# Test connection
redis-cli ping  # Should return PONG
```

### Migration Issues

**Error**: `Dirty database version X`
```bash
# Force version (if migration was interrupted)
migrate -path ./migrations -database "postgres://..." force <version-number>

# Then run migrations again
migrate -path ./migrations -database "postgres://..." up
```

### AI Scoring Not Working

**Error**: `AI scoring timed out` or `Groq API error`
```bash
# Check Groq API key is valid
curl https://api.groq.com/openai/v1/models \
  -H "Authorization: Bearer $GROQ_API_KEY"

# Check worker is running
ps aux | grep worker

# Check worker logs
docker-compose logs worker  # If using Docker

# Check Redis queue
redis-cli
> XLEN ai-scoring  # Should show pending jobs
> XREAD STREAMS ai-scoring 0  # View pending jobs
```

### Email Not Sending

**Error**: `SMTP connection failed`
```bash
# Check MailHog/MailCatcher is running
curl http://localhost:1080  # MailCatcher
curl http://localhost:8025  # MailHog

# Check SMTP settings in .env
# For local dev, use MailHog/MailCatcher (no auth required)
SMTP_HOST=localhost
SMTP_PORT=1025  # MailCatcher
# or
SMTP_PORT=1025  # MailHog

SMTP_USER=
SMTP_PASS=
```

---

## Next Steps

- **Add more tests**: Experiment with multiple questions, retake policies
- **Test AI scoring**: Submit various quality answers to see score variations
- **Manual review flow**: Practice reviewer workflow
- **Run integration tests**: `go test ./tests/integration/...`
- **Read API docs**: Check `specs/001-assessment-system-baseline/contracts/openapi.yaml`
- **Explore codebase**: Follow Clean Architecture layers in `internal/`

---

## Useful Commands Cheat Sheet

```bash
# Start everything (Docker)
docker-compose up -d

# View logs
docker-compose logs -f api
docker-compose logs -f worker

# Stop everything
docker-compose down

# Reset database (CAUTION: deletes all data)
docker-compose down -v
docker-compose up -d

# Run tests
go test ./...

# Check coverage
go test -cover ./...

# Format code
go fmt ./...

# Run linter
golangci-lint run

# Database shell
psql -h localhost -U assessly -d assessly_dev

# Redis shell
redis-cli

# View migrations
migrate -path ./migrations -database "postgres://..." version

# Health check
curl http://localhost:8080/health
```

---

## Support

For issues or questions:
1. Check [troubleshooting section](#troubleshooting) above
2. Review logs: `docker-compose logs` or check console output
3. Verify .env configuration matches your environment
4. Consult feature spec: `specs/001-assessment-system-baseline/spec.md`
5. Review API contracts: `specs/001-assessment-system-baseline/contracts/openapi.yaml`

Happy coding! 🚀
