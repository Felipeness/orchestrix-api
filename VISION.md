# Orchestrix

**Plataforma de Observabilidade + Auto-Remediação**

## Visão

Orchestrix combina observabilidade (como Datadog) com automação de workflows (como Temporal) para criar uma plataforma unificada de monitoramento e resposta automática a incidentes.

```
┌─────────────────────────────────────────────────────────────────┐
│                        ORCHESTRIX                                │
├─────────────────────┬───────────────────┬───────────────────────┤
│   OBSERVABILIDADE   │     DETECÇÃO      │      AUTOMAÇÃO        │
├─────────────────────┼───────────────────┼───────────────────────┤
│ • Métricas          │ • Regras/Alertas  │ • Workflows Temporal  │
│ • Logs              │ • Anomalias       │ • Auto-remediação     │
│ • Traces            │ • Thresholds      │ • Runbooks automáticos│
│ • Events            │ • ML/Patterns     │ • Notificações        │
└─────────────────────┴───────────────────┴───────────────────────┘
         ▲                    │                      │
         │                    ▼                      ▼
    [Agents]           [Alert Engine]         [Temporal Workers]
```

## Diferencial

| Produto | Foco | Limitação |
|---------|------|-----------|
| **Datadog** | Observabilidade | Automação limitada |
| **PagerDuty** | Alertas + incident response | Sem observabilidade nativa |
| **Shoreline.io** | Auto-remediação | Focado apenas em remediation |
| **StackStorm** | Event-driven automation | Sem observabilidade |
| **Rundeck** | Runbook automation | Manual, não reativo |

**Orchestrix** = Observabilidade + Detecção + Automação em uma plataforma integrada.

## Arquitetura

```
orchestrix/
├── orchestrix-api/      # API REST (Go + Chi)
├── orchestrix-bff/      # Backend for Frontend (TypeScript)
├── orchestrix-web/      # Frontend (React)
└── orchestrix-worker/   # Temporal Workers (Go)
```

### Stack Tecnológica

- **Backend**: Go, Chi, sqlc
- **Workflow Engine**: Temporal
- **Auth**: Keycloak (OIDC)
- **Database**: PostgreSQL (RLS multi-tenant)
- **Frontend**: React, TypeScript
- **Infra**: Docker, Kubernetes-ready

## Status Atual

### Implementado

- [x] API RESTful com CRUD de workflows e execuções
- [x] Worker Temporal com activities (Validate, Process, Notify)
- [x] Autenticação OIDC com Keycloak
- [x] Multi-tenancy com Row-Level Security
- [x] Execução async com callback de status
- [x] Estrutura de Alerts e Audit Logs
- [x] BFF estruturado (TypeScript)
- [x] Frontend base (React)

### Roadmap

#### Fase 1: Alert → Workflow Trigger
- [ ] Conectar alertas com execução automática de workflows
- [ ] Regras configuráveis (threshold, pattern matching)
- [ ] Webhook para receber eventos externos

#### Fase 2: Ingestão de Dados
- [ ] Endpoint para receber métricas (Prometheus format)
- [ ] Endpoint para receber logs (structured JSON)
- [ ] Endpoint para receber traces (OpenTelemetry)
- [ ] Storage time-series (InfluxDB ou ClickHouse)

#### Fase 3: Workflow Definition Interpreter
- [ ] Parser de JSON definition para steps dinâmicos
- [ ] Activities reutilizáveis:
  - HTTP Request
  - SSH Command
  - Kubernetes (scale, restart)
  - AWS/GCP/Azure actions
  - Database queries
- [ ] Condicionais e loops no workflow
- [ ] Variables e templating

#### Fase 4: Integrações
- [ ] Cloud Providers (AWS, GCP, Azure)
- [ ] Kubernetes API
- [ ] Messaging (Slack, Teams, Discord)
- [ ] Incident Management (PagerDuty, Opsgenie)
- [ ] Databases (PostgreSQL, MySQL, Redis)

#### Fase 5: Observabilidade Avançada
- [ ] Dashboards customizáveis
- [ ] Correlação de métricas/logs/traces
- [ ] Anomaly detection (ML-based)
- [ ] Service maps e dependency graphs

#### Fase 6: Enterprise Features
- [ ] SSO avançado (SAML, LDAP)
- [ ] RBAC granular
- [ ] Compliance reports
- [ ] Data retention policies
- [ ] High availability setup

## Exemplo de Uso

### Cenário: Auto-scaling baseado em métricas

1. **Ingestão**: Agent envia métrica `cpu_usage = 95%`
2. **Detecção**: Alert rule detecta `cpu_usage > 90%`
3. **Automação**: Workflow executa:
   - Notify Slack: "High CPU detected"
   - Scale up Kubernetes deployment
   - Wait 5 minutes
   - Verify metrics normalized
   - Notify: "Issue resolved"

### Cenário: Log-based remediation

1. **Ingestão**: App envia log `ERROR: Database connection timeout`
2. **Detecção**: Pattern matching identifica erro de DB
3. **Automação**: Workflow executa:
   - Check database health
   - Restart connection pool
   - If still failing: failover to replica
   - Create incident ticket
   - Notify on-call engineer

## Contribuindo

```bash
# Clone
git clone https://github.com/orchestrix/orchestrix.git

# Start infrastructure
docker-compose up -d

# Run API
cd orchestrix-api && go run cmd/api/main.go

# Run Worker
cd orchestrix-api && go run cmd/worker/main.go
```

## Licença

MIT
