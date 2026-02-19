---
name: observability
description: Add logging, tracing, or metrics to code in this project. Use when asked to instrument a service, add a log line, record a metric, create a span, or wire up observability to a new component.
---

## Overview

Three pillars, all behind interfaces in `pkg/observability/`:

| Pillar | Interface | Implementation |
|---|---|---|
| Logging | `observability.Logger` | Uber Zap |
| Tracing | `observability.Tracer` / `observability.Span` | OpenTelemetry → OTLP gRPC |
| Metrics | `observability.Meter` | Prometheus |

Observability is initialised once in `cmd/server/main.go` and passed down via constructor injection. Never use global loggers or metrics.

---

## Logging

### Interface

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    Fatal(msg string, fields ...Field)   // calls os.Exit(1)
    With(fields ...Field) Logger         // returns child logger with pre-set fields
}
```

### Field helpers (`pkg/observability/field.go`)

```go
observability.String("key", "value")
observability.Int("key", 42)
observability.Err(err)           // key is always "error"
```

### Usage patterns

```go
// Plain log
log.Info("server started")

// With structured fields
log.Info("user created", observability.String("user_id", strconv.FormatInt(id, 10)))

// Error
log.Error("failed to connect", observability.Err(err))

// Fatal (terminates process)
log.Fatal("cannot initialise database", observability.Err(err))

// Child logger with shared context (e.g. per-request)
reqLog := log.With(observability.String("request_id", reqId))
reqLog.Info("processing request")
reqLog.Error("request failed", observability.Err(err))
```

### When to use each level

| Level | When |
|---|---|
| `Debug` | Verbose diagnostics, not emitted in production |
| `Info` | Normal lifecycle events (started, stopped, request handled) |
| `Warn` | Recoverable unexpected state |
| `Error` | Operation failed but process continues |
| `Fatal` | Unrecoverable startup failure; calls `os.Exit(1)` |

---

## Tracing

### Interface

```go
type Tracer interface {
    Start(ctx context.Context, name string) (context.Context, Span)
}

type Span interface {
    End()
    RecordError(err error)
}
```

### Usage pattern

```go
func (s *fooService) CreateFoo(ctx context.Context, foo *model.Foo) (*model.Foo, error) {
    ctx, span := s.tracer.Start(ctx, "FooService.CreateFoo")
    defer span.End()

    result, err := doWork(ctx)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }
    return result, nil
}
```

- Always `defer span.End()` immediately after `Start`.
- Call `span.RecordError(err)` before returning errors — do not omit this.
- Pass the returned `ctx` to any downstream calls so child spans nest correctly.
- Name spans as `"<Type>.<Method>"`, e.g. `"UserService.GetUser"`, `"UserRepository.Get"`.

### Injecting the tracer

Add `tracer observability.Tracer` to the struct and pass via constructor:

```go
type fooService struct {
    uowFactory repository.UnitOfWorkFactory
    snowflake   snowflake.Snowflake
    tracer      observability.Tracer
}

func NewFooService(uowFactory repository.UnitOfWorkFactory, sf snowflake.Snowflake, tracer observability.Tracer) FooService {
    return &fooService{uowFactory: uowFactory, snowflake: sf, tracer: tracer}
}
```

Wire in `cmd/server/main.go`:
```go
fooSvc := service.NewFooService(dbs.UnitOfWorkFactory, idGen, obs.Tracer())
```

---

## Metrics

### Four metric types

```go
// Counter — monotonically increasing
counter := meter.Counter("requests_total", observability.MetricOpt{
    Help:      "Total requests",
    LabelKeys: []string{"method", "status"},
})
counter.Inc(1, observability.Label{Key: "method", Value: "create"}, observability.Label{Key: "status", Value: "ok"})

// Histogram — distribution of values
hist := meter.Histogram("request_size_bytes", observability.MetricOpt{
    Help:    "Request payload size",
    Buckets: []float64{64, 256, 1024, 4096},
})
hist.Observe(float64(len(body)))

// Gauge — point-in-time value
gauge := meter.Gauge("queue_depth", observability.MetricOpt{Help: "Current queue depth"})
gauge.Set(float64(len(queue)))
gauge.Add(1)

// Timer — records duration in seconds automatically
timer := meter.Timer("operation_duration_seconds", observability.MetricOpt{
    Help:      "Operation latency",
    LabelKeys: []string{"operation"},
})
stop := timer.Start(observability.Label{Key: "operation", Value: "create"})
defer stop()   // records duration when function returns
```

### MetricOpt fields

```go
observability.MetricOpt{
    Help:        "Human-readable description",       // required
    Buckets:     []float64{0.001, 0.01, 0.1, 1},   // histogram only
    LabelKeys:   []string{"status", "method"},       // dynamic labels per observation
    ConstLabels: []observability.Label{              // static labels on all observations
        {Key: "service", Value: "foo"},
    },
}
```

### Injecting the meter

```go
type fooService struct {
    // ...
    requestsTotal observability.Counter
    latency       observability.Timer
}

func NewFooService(/* ... */, meter observability.Meter) FooService {
    return &fooService{
        // ...
        requestsTotal: meter.Counter("foo_requests_total", observability.MetricOpt{
            Help:      "Total foo service requests",
            LabelKeys: []string{"operation", "status"},
        }),
        latency: meter.Timer("foo_operation_duration_seconds", observability.MetricOpt{
            Help:      "Foo service operation latency",
            LabelKeys: []string{"operation"},
        }),
    }
}
```

Wire in `cmd/server/main.go`:
```go
fooSvc := service.NewFooService(dbs.UnitOfWorkFactory, idGen, obs.Meter())
```

---

## GORM metrics (automatic)

The `GormMetricsPlugin` is registered in `bootstrap.InitializeDatabase` and automatically tracks:

| Metric | Type | Labels |
|---|---|---|
| `gorm_query_duration_seconds` | Histogram | `operation` (create/query/update/delete/row/raw) |
| `gorm_query_total` | Counter | `operation` |
| `gorm_query_errors_total` | Counter | `operation` |

No manual instrumentation needed for database calls.

---

## Adding observability to an existing service

1. Add `logger observability.Logger` / `tracer observability.Tracer` / metric fields to the struct.
2. Initialise metrics in the constructor with `meter.Counter(...)` etc.
3. Add spans at the start of each public method with `defer span.End()`.
4. Log meaningful lifecycle events (created, not found, error).
5. Record metric values at the appropriate points.
6. Update `cmd/server/main.go` to pass `obs.Logger()`, `obs.Tracer()`, `obs.Meter()` to the constructor.

---

## Initialization reference (`cmd/server/main.go`)

```go
cfg := implementation.Config{ServiceName: "go-grpc-template"}
obs, err := implementation.NewObservability(cfg)   // creates logger + meter + tracer

log := obs.Logger()
if err := obs.Start(ctx); err != nil {             // starts :9090 /metrics HTTP server
    log.Error("failed to start observability", observability.Err(err))
}
defer func() {
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _ = obs.Close(shutdownCtx)                     // flushes traces, stops metrics server
}()

// Pass to database (for GORM plugin)
dbs, err := bootstrap.InitializeDatabase(dsn, obs.Meter())

// Pass to services
svc := service.NewFooService(dbs.UnitOfWorkFactory, idGen, obs.Tracer(), obs.Meter())
```