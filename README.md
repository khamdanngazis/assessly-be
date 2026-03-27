# Assessly Backend

A flexible assessment platform for creating and managing essay-based tests with AI-powered automatic scoring and manual review capabilities.

## Overview

Assessly enables educators, HR professionals, and trainers to create essay-based tests that participants can access without authentication. The system provides:

- **Test Management**: Create, edit, and publish tests with multiple essay questions
- **Anonymous Participation**: Participants access tests via secure links without account creation
- **AI Scoring**: Automatic scoring using Groq AI models
- **Manual Review**: Human reviewers can view and override AI assessments
- **Clean Architecture**: Built with Go following Clean Architecture principles

## Quick Start

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 14+
- Redis 7+
- Docker & Docker Compose (for containerized deployment)

### Docker Deployment (Recommended)

**Architecture**: API and Worker run in Docker containers, PostgreSQL and Redis run on host machine.

```bash
# 1. Setup PostgreSQL on host
sudo apt-get install postgresql-14  # Ubuntu
createdb assessly_dev
psql assessly_dev < migrations/schema.sql

# 2. Setup Redis on host
sudo apt-get install redis-server
sudo systemctl start redis

# 3. Configure environment
cp .env.example .env
# Edit .env: Set DB_PASSWORD, REDIS_PASSWORD, JWT_SECRET, GROQ_API_KEY

# 4. Build and run containers
docker-compose build
docker-compose up -d

# 5. Check health
curl http://localhost:8080/health
```

📖 **Full deployment guide**: See [DEPLOYMENT.md](DEPLOYMENT.md)  
⚡ **Quick reference**: See [QUICKSTART.md](QUICKSTART.md)  
📋 **Architecture summary**: See [DEPLOYMENT_SUMMARY.md](DEPLOYMENT_SUMMARY.md)

### Manual Setup (Development)

```bash
# Install dependencies
go mod download

# Setup database
export DATABASE_URL="postgres://user:pass@localhost:5432/assessly_dev?sslmode=disable"
migrate -path migrations -database $DATABASE_URL up

# Run API server
go run cmd/api/main.go

# Run worker (separate terminal)
go run cmd/worker/main.go
```

## Project Structure

```
assessly-be/
├── cmd/
│   ├── api/              # HTTP API server
│   └── worker/           # Async worker for AI scoring
├── internal/
│   ├── domain/           # Core business entities
│   ├── usecase/          # Business logic
│   ├── delivery/         # HTTP handlers, workers
│   └── infrastructure/   # External dependencies
├── migrations/           # Database migrations
├── tests/                # Unit, integration, contract tests
├── specs/                # Feature specifications
└── docs/                 # Architecture decision records
```

## Development

### Build

```bash
# Build all binaries
make build

# Build specific component
go build -o bin/api cmd/api/main.go
go build -o bin/worker cmd/worker/main.go
```

### Testing

```bash
# Run all tests (141 tests)
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test suites
go test ./tests/unit/...          # 56 unit tests
go test ./tests/integration/...   # 41 integration tests
go test ./tests/contract/...      # 44 contract tests
```

### Database Migrations

```bash
# Run migrations
migrate -path migrations -database "$DATABASE_URL" up

# Rollback one migration
migrate -path migrations -database "$DATABASE_URL" down 1

# Check migration version
migrate -path migrations -database "$DATABASE_URL" version

# Create new migration
migrate create -ext sql -dir migrations -seq add_user_status
```

### Code Quality

```bash
# Format code
gofmt -w .

# Run linter
golangci-lint run

# Run security scanner
gosec ./...
```

## Deployment

### Architecture

```
┌─────────────────────────────────────────────┐
│              Host Machine                    │
│                                              │
│  ┌─────────────────┐   ┌─────────────────┐ │
│  │  PostgreSQL 14+ │   │    Redis 7+     │ │
│  │   Port 5432     │   │   Port 6379     │ │
│  └────────▲────────┘   └────────▲────────┘ │
│           │                     │           │
│  ┌────────┴─────────────────────┴────────┐ │
│  │        Docker Bridge Network          │ │
│  │                                        │ │
│  │  ┌──────────────┐  ┌──────────────┐  │ │
│  │  │  API Server  │  │    Worker    │  │ │
│  │  │  Port 8080   │  │  AI Scoring  │  │ │
│  │  └──────────────┘  └──────────────┘  │ │
│  └────────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
```

### Deployment Guides

- **Comprehensive Guide**: [DEPLOYMENT.md](DEPLOYMENT.md) - Full deployment instructions with PostgreSQL/Redis setup, troubleshooting, and production checklist
- **Quick Reference**: [QUICKSTART.md](QUICKSTART.md) - Fast deployment for developers
- **Architecture Summary**: [DEPLOYMENT_SUMMARY.md](DEPLOYMENT_SUMMARY.md) - Technical reference for operations

### Key Points

- PostgreSQL and Redis run on **host machine** (not in containers)
- API and Worker run in **Docker containers**
- Containers connect to host services via `host.docker.internal`
- Linux users: Requires Docker 20.10+ with host-gateway support

### Quick Deploy

```bash
# 1. Setup databases on host (see DEPLOYMENT.md)
createdb assessly_dev
psql assessly_dev < migrations/schema.sql
sudo systemctl start redis

# 2. Configure environment
cp .env.example .env
# Edit: DB_PASSWORD, REDIS_PASSWORD, JWT_SECRET, GROQ_API_KEY

# 3. Deploy
docker-compose build
docker-compose up -d

# 4. View logs
docker-compose logs -f api
docker-compose logs -f worker

# 5. Scale workers
docker-compose up -d --scale worker=3
```

## API Documentation

API endpoints are documented in OpenAPI 3.0 format:
- **Contract**: [specs/001-assessment-system-baseline/contracts/openapi.yaml](specs/001-assessment-system-baseline/contracts/openapi.yaml)

### Key Endpoints

- `POST /api/v1/auth/register` - Register creator/reviewer account
- `POST /api/v1/auth/login` - Authenticate and get JWT token
- `POST /api/v1/tests` - Create new test
- `POST /api/v1/tests/:id/questions` - Add questions to test
- `POST /api/v1/tests/:id/publish` - Publish test for participants
- `POST /api/v1/submissions/access` - Request test access token
- `POST /api/v1/submissions` - Submit test answers
- `PUT /api/v1/reviews/:answerId` - Add manual review

## Architecture

Built with **Clean Architecture** principles:

- **Domain Layer**: Core business entities and interfaces
- **Use Case Layer**: Application business logic
- **Delivery Layer**: HTTP handlers, workers
- **Infrastructure Layer**: Database, external APIs, message queues

### Tech Stack

- **Language**: Go 1.21+
- **Web Framework**: chi router
- **Database**: PostgreSQL with pgx driver
- **Cache/Queue**: Redis Streams
- **AI**: Groq API
- **Auth**: JWT with golang-jwt
- **Migrations**: golang-migrate
- **Testing**: testify
- **Observability**: slog, Prometheus, OpenTelemetry

## Configuration

All configuration via environment variables. See [.env.example](.env.example) for required variables.

Key configurations:
- `DB_*`: PostgreSQL connection
- `REDIS_*`: Redis connection
- `JWT_SECRET`: JWT signing key
- `GROQ_API_KEY`: Groq AI API key
- `SMTP_*`: Email service configuration

## Contributing

1. Clone the repository
2. Create feature branch: `git checkout -b feature/your-feature`
3. Make changes following Clean Architecture
4. Add tests (minimum 70% coverage)
5. Run linter: `make lint`
6. Submit pull request

## Documentation

- **Feature Spec**: [specs/001-assessment-system-baseline/spec.md](specs/001-assessment-system-baseline/spec.md)
- **Implementation Plan**: [specs/001-assessment-system-baseline/plan.md](specs/001-assessment-system-baseline/plan.md)
- **Data Model**: [specs/001-assessment-system-baseline/data-model.md](specs/001-assessment-system-baseline/data-model.md)
- **Quickstart Guide**: [specs/001-assessment-system-baseline/quickstart.md](specs/001-assessment-system-baseline/quickstart.md)
- **Tasks**: [specs/001-assessment-system-baseline/tasks.md](specs/001-assessment-system-baseline/tasks.md)

## License

Private - All rights reserved

---

**Status**: In Development  
**Version**: 0.1.0  
**Feature Branch**: 001-assessment-system-baseline