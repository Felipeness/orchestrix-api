# Orchestrix API

**Core backend** da plataforma Orchestrix AIOps - o cérebro que processa métricas, detecta anomalias, executa workflows e gerencia incidentes.

## O que é o Orchestrix?

Orchestrix é uma plataforma AIOps completa que **substitui Grafana + Datadog + Prometheus + PagerDuty** e adiciona inteligência artificial para entender, explicar e resolver problemas automaticamente.

> Veja [VISION.md](../VISION.md) para a visão completa do projeto.

## Responsabilidades do API

```
┌─────────────────────────────────────────────────────────────────┐
│                        ORCHESTRIX API                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  OBSERVABILITY           INTELLIGENCE          ACTION           │
│  ─────────────           ────────────          ──────           │
│  • Metrics ingestion     • Anomaly detection   • Temporal       │
│  • Log aggregation       • Correlation         • Workflows      │
│  • Trace collection      • Root cause          • Auto-remediate │
│  • Alert rules           • LLM analysis        • Notifications  │
│                                                                  │
│  INCIDENT MANAGEMENT                                            │
│  ────────────────────                                           │
│  • On-call scheduling    • Escalation policies                  │
│  • Incident lifecycle    • SLA tracking                         │
│  • Status pages          • Post-mortems                         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Tech Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Language | Go 1.22+ | Performance, concurrency |
| HTTP | Chi router | Lightweight, composable |
| Database | PostgreSQL 16 | Metadata, configs |
| Time-series | TimescaleDB | Metrics storage (planned) |
| Logs | ClickHouse | Log aggregation (planned) |
| ORM | sqlc | Type-safe queries |
| Workflows | Temporal | Durable execution |
| Auth | Keycloak | Identity, multi-tenant |

## Project Structure (Hexagonal Architecture)

```
orchestrix-api/
├── cmd/
│   ├── api/                          # HTTP API server
│   └── worker/                       # Temporal worker
├── internal/
│   ├── adapter/
│   │   ├── driven/                   # Secondary adapters (outbound)
│   │   │   ├── postgres/             # Database repositories
│   │   │   └── temporal/             # Workflow execution
│   │   └── driving/                  # Primary adapters (inbound)
│   │       └── http/                 # HTTP handlers
│   ├── core/
│   │   ├── domain/                   # Domain models & errors
│   │   ├── port/                     # Interfaces (ports)
│   │   └── service/                  # Business logic
│   ├── auth/                         # Keycloak middleware
│   └── db/                           # sqlc generated code
├── pkg/
│   ├── apperror/                     # Centralized error handling
│   ├── validation/                   # Input validation utilities
│   ├── observability/                # Logging, metrics, tracing
│   ├── temporal/                     # Temporal client utilities
│   └── database/                     # Database utilities
├── api/openapi/v1/                   # OpenAPI specification
├── sql/
│   ├── migrations/                   # Database migrations
│   └── queries/                      # sqlc query definitions
└── keycloak/
    └── realm-*.json                  # Keycloak realm config
```

## Current Status

### Implemented
- [x] **Hexagonal Architecture** - Ports & Adapters pattern
- [x] Multi-tenant authentication (Keycloak)
- [x] Workflow CRUD operations
- [x] Workflow execution via Temporal
- [x] Dynamic workflow engine
- [x] Alert rules with threshold conditions
- [x] Alert triggering from rules
- [x] Auto-remediation via workflows
- [x] Audit logging
- [x] OpenAPI 3.1 documentation
- [x] **Centralized error handling** (pkg/apperror)
- [x] **Input validation** (pkg/validation)
- [x] **Observability package** (pkg/observability)
- [x] Unit tests for services
- [x] Integration tests for repositories

### In Progress
- [ ] Metrics ingestion API (Prometheus-compatible)
- [ ] TimescaleDB integration
- [ ] Anomaly detection engine

### Planned (Phase 1-6)
- [ ] Log aggregation (ClickHouse)
- [ ] Distributed tracing (OTEL)
- [ ] LLM integration (Claude/GPT)
- [ ] On-call scheduling
- [ ] Escalation policies
- [ ] Status pages
- [ ] Incident management

## API Endpoints

### Core APIs

```
Health
├── GET  /health              # Health check
├── GET  /health/live         # Liveness probe
└── GET  /health/ready        # Readiness probe

Workflows
├── GET    /api/v1/workflows           # List workflows
├── POST   /api/v1/workflows           # Create workflow
├── GET    /api/v1/workflows/:id       # Get workflow
├── PUT    /api/v1/workflows/:id       # Update workflow
├── DELETE /api/v1/workflows/:id       # Delete workflow
└── POST   /api/v1/workflows/:id/execute  # Execute workflow

Executions
├── GET  /api/v1/executions            # List executions
├── GET  /api/v1/executions/:id        # Get execution
└── POST /api/v1/executions/:id/cancel # Cancel execution

Alerts
├── GET  /api/v1/alerts                # List alerts
├── POST /api/v1/alerts                # Create alert
├── GET  /api/v1/alerts/:id            # Get alert
├── POST /api/v1/alerts/:id/acknowledge  # Acknowledge
└── POST /api/v1/alerts/:id/resolve      # Resolve

Alert Rules
├── GET  /api/v1/alert-rules           # List rules
├── POST /api/v1/alert-rules           # Create rule
├── GET  /api/v1/alert-rules/:id       # Get rule
├── PUT  /api/v1/alert-rules/:id       # Update rule
└── DELETE /api/v1/alert-rules/:id     # Delete rule

Audit Logs
├── GET  /api/v1/audit-logs            # List audit logs
└── GET  /api/v1/audit-logs/:id        # Get audit log
```

### Planned APIs

```
Metrics (Phase 1)
├── POST /api/v1/metrics/write         # Prometheus remote write
├── GET  /api/v1/metrics/query         # PromQL query
└── GET  /api/v1/metrics/labels        # Label values

Logs (Phase 2)
├── POST /api/v1/logs/ingest           # Log ingestion
└── GET  /api/v1/logs/query            # Log search

Incidents (Phase 4)
├── GET  /api/v1/incidents             # List incidents
├── POST /api/v1/incidents             # Create incident
├── GET  /api/v1/incidents/:id/timeline  # Incident timeline
└── POST /api/v1/incidents/:id/resolve   # Resolve incident

On-Call (Phase 4)
├── GET  /api/v1/oncall/schedules      # List schedules
├── POST /api/v1/oncall/schedules      # Create schedule
└── GET  /api/v1/oncall/current        # Who's on call now
```

## Getting Started

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- PostgreSQL 16
- Temporal server

### Development

```bash
# Start dependencies
docker-compose up -d postgres temporal keycloak

# Run API server
go run ./cmd/api

# Run Temporal worker
go run ./cmd/worker

# Generate sqlc code
sqlc generate

# Run tests
go test ./...
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | API server port | `8080` |
| `DATABASE_URL` | PostgreSQL connection | required |
| `TEMPORAL_HOST` | Temporal server | `localhost:7233` |
| `TEMPORAL_TASK_QUEUE` | Task queue name | `orchestrix-queue` |
| `KEYCLOAK_URL` | Keycloak server | `http://localhost:8180` |
| `KEYCLOAK_REALM` | Keycloak realm | `orchestrix` |

## Architecture Principles

1. **Multi-tenant by default** - Row-level security, tenant isolation
2. **Event-driven** - Temporal for durable workflows
3. **Type-safe** - sqlc for database, strong Go typing
4. **Observable** - Structured logging (slog), metrics
5. **Secure** - Keycloak auth, audit trail

## License

MIT
