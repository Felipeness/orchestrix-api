# Metrics Ingestion System

Sistema de ingestão de métricas customizado que **substitui Prometheus/Datadog/Grafana** com uma solução nativa usando TimescaleDB.

## Arquitetura

```
┌─────────────────────────────────────────────────────────────────┐
│                     METRICS INGESTION                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  INGESTION              STORAGE               QUERY             │
│  ─────────              ───────               ─────             │
│  • POST /ingest         • TimescaleDB         • Time range      │
│  • POST /ingest/batch   • Hypertables         • Aggregation     │
│  • JSON format          • Compression         • Time buckets    │
│  • Multi-tenant         • Retention           • Label filters   │
│                                                                  │
│  ALERTING                                                       │
│  ────────                                                       │
│  • Threshold evaluation on ingestion                            │
│  • Integration with AlertRule system                            │
│  • Auto-remediation via Temporal workflows                      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## API Endpoints

### Ingestion

```bash
# Single metric
POST /api/v1/metrics/ingest
{
    "name": "cpu_usage",
    "value": 75.5,
    "labels": {"host": "server-1", "region": "us-east"},
    "source": "agent",
    "timestamp": "2025-01-15T10:30:00Z"
}

# Batch ingestion (max 10k metrics)
POST /api/v1/metrics/ingest/batch
{
    "metrics": [
        {"name": "cpu_usage", "value": 75.5, "labels": {"host": "server-1"}},
        {"name": "memory_usage", "value": 60.2, "labels": {"host": "server-1"}}
    ]
}
```

### Query

```bash
# Query metrics with filters
GET /api/v1/metrics?name=cpu_usage&start=2025-01-15T00:00:00Z&end=2025-01-15T23:59:59Z

# List metric names
GET /api/v1/metrics/names?prefix=cpu

# Latest value
GET /api/v1/metrics/latest/{name}

# Aggregated stats (count, avg, min, max, sum, p50, p95, p99)
GET /api/v1/metrics/aggregate/{name}?start=...&end=...

# Time-bucketed series
GET /api/v1/metrics/series/{name}?bucket=5m&start=...&end=...
```

### Metric Definitions

```bash
# List definitions
GET /api/v1/metrics/definitions

# Create definition
POST /api/v1/metrics/definitions
{
    "name": "cpu_usage",
    "display_name": "CPU Usage",
    "description": "CPU utilization percentage",
    "unit": "%",
    "type": "gauge",
    "aggregation": "avg",
    "retention_days": 30,
    "alert_threshold": {
        "warning": 70,
        "critical": 90
    }
}

# Get/Update/Delete
GET    /api/v1/metrics/definitions/{name}
PUT    /api/v1/metrics/definitions/{name}
DELETE /api/v1/metrics/definitions/{name}
```

## TimescaleDB Features

### Hypertable

A tabela `metrics` é convertida em hypertable com chunks de 1 dia:

```sql
SELECT create_hypertable('metrics', 'timestamp', chunk_time_interval => INTERVAL '1 day');
```

### Continuous Aggregates

Rollups horários pré-computados para queries rápidas:

```sql
CREATE MATERIALIZED VIEW metrics_hourly
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', timestamp) AS bucket,
    tenant_id, name,
    count(*), avg(value), min(value), max(value), sum(value)
FROM metrics
GROUP BY bucket, tenant_id, name;
```

### Compression

Chunks maiores que 7 dias são comprimidos automaticamente:

```sql
ALTER TABLE metrics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'tenant_id, name'
);

SELECT add_compression_policy('metrics', INTERVAL '7 days');
```

### Retention

Política de retenção padrão de 30 dias:

```sql
SELECT add_retention_policy('metrics', INTERVAL '30 days');
```

## Tipos de Métricas

| Type | Description | Aggregation |
|------|-------------|-------------|
| `gauge` | Valor instantâneo (CPU, memória) | avg |
| `counter` | Sempre crescente (requests, bytes) | rate |
| `histogram` | Distribuição (latência) | percentile |
| `summary` | Estatísticas pré-calculadas | avg |

## Integração com Alertas

Quando uma métrica é ingerida, o sistema avalia automaticamente:

1. **Threshold Check**: Compara valor com `alert_threshold` da definição
2. **Rule Evaluation**: Executa regras de alerta configuradas
3. **Auto-remediation**: Dispara workflow Temporal se configurado

```go
// Async alert evaluation (não bloqueia ingestion)
go s.evaluateAlerts(ctx, tenantID, metricName, value)
```

## Performance

- **Batch insert**: Usa `pgx.CopyFrom` para alto throughput (10k+ metrics/sec)
- **Async alerts**: Avaliação de alertas não bloqueia ingestion
- **Index otimizado**: Índices em (tenant_id, name, timestamp)
- **Compression**: Reduz storage em ~90% após 7 dias

## Hexagonal Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP Handler                              │
│                   (internal/adapter/driving/http/metric.go)      │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                       MetricService                              │
│                   (internal/core/service/metric.go)              │
│                                                                  │
│  • Ingest / IngestBatch                                         │
│  • Query / GetLatest / GetAggregate / GetSeries                 │
│  • ListNames / ListDefinitions                                  │
│  • CreateDefinition / UpdateDefinition / DeleteDefinition       │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    MetricRepository (port)                       │
│                   (internal/core/port/secondary.go)              │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                  PostgreSQL Repository                           │
│               (internal/adapter/driven/postgres/metric.go)       │
│                                                                  │
│  • Save / SaveBatch (pgx CopyFrom)                              │
│  • FindByQuery / CountByQuery                                   │
│  • FindLatest / GetAggregate / GetSeries                        │
│  • ListNames                                                    │
└─────────────────────────────────────────────────────────────────┘
```

## Multi-tenant

Todas as queries são filtradas por `tenant_id` automaticamente via:

```go
func (s *MetricService) Ingest(ctx context.Context, input port.IngestMetricInput) error {
    if err := s.tenantSetter.SetTenantContext(ctx, input.TenantID); err != nil {
        return err
    }
    // ... rest of the logic
}
```
