# Orchestrix API

Core backend service for the Orchestrix platform - workflow orchestration, monitoring, and async alerts at scale.

## Tech Stack

- **Language**: Go 1.22+
- **HTTP Router**: chi
- **Database**: PostgreSQL 16 (via sqlc)
- **Workflow Engine**: Temporal
- **Messaging**: Kafka (AWS MSK)

## Project Structure

```
orchestrix-api/
├── cmd/
│   ├── api/           # HTTP API entrypoint
│   └── worker/        # Temporal worker entrypoint
├── internal/          # Private application code (group-by-feature)
│   ├── workflow/      # Temporal workflows
│   ├── activity/      # Temporal activities
│   ├── execution/     # Execution tracking
│   ├── tenant/        # Multi-tenancy (RLS, context)
│   ├── auth/          # Authentication (Keycloak)
│   ├── audit/         # Audit trail
│   ├── alert/         # Alerting system
│   └── shared/        # Shared types (minimal)
├── pkg/               # Public libraries
│   ├── temporal/      # Temporal utilities
│   └── database/      # Database utilities
└── sql/
    ├── migrations/    # Database migrations
    └── queries/       # sqlc queries
```

## Getting Started

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Temporal CLI (optional)

### Development

```bash
# Install dependencies
go mod download

# Run API server
go run ./cmd/api

# Run Temporal worker
go run ./cmd/worker

# Run tests
go test ./...

# Generate sqlc code
sqlc generate
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | API server port | `8080` |
| `TEMPORAL_HOST` | Temporal server address | `localhost:7233` |
| `TEMPORAL_TASK_QUEUE` | Temporal task queue name | `orchestrix-queue` |
| `DATABASE_URL` | PostgreSQL connection string | - |

## API Endpoints

### Health

- `GET /health` - Health check
- `GET /health/live` - Liveness probe
- `GET /health/ready` - Readiness probe

### API v1

- `GET /api/v1` - API info

## Architecture

This service follows:
- **Group-by-feature** package organization (Go community consensus)
- **Temporal** for durable workflow execution
- **sqlc** for type-safe database queries
- **Structured logging** with slog

## License

MIT
