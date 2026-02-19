# go-grpc-template

A Go gRPC template with common reliability patterns.

Most gRPC examples stop at getting a server running. This template adds the patterns that tend to come up once you start building real services: idempotent writes, circuit breaking, retry with backoff, distributed ID generation, and structured observability — all wired together so you can start from something closer to what you'd actually ship.

## What's Included

**Reliability**
- Idempotency — prevent duplicate writes using a request ID + result cache
- Circuit breaker — wraps downstream calls with open/half-open/closed state
- Retry with exponential backoff

**Observability**
- Structured logging via [Zap](https://github.com/uber-go/zap)
- Metrics via Prometheus (with gRPC server metrics and GORM query metrics)
- Distributed tracing via OpenTelemetry

**Infrastructure**
- Snowflake-based distributed ID generation
- PostgreSQL with GORM and a Unit of Work pattern
- Database migrations via [golang-migrate](https://github.com/golang-migrate/migrate)
- gRPC health check endpoint with live DB ping
- Graceful shutdown

**Developer Experience**
- Unit and integration tests (integration tests use Docker via testcontainers)
- Error interceptor that maps domain errors to gRPC status codes
- [Claude Code skills](.claude/skills/) — context-aware prompts for adding features, writing tests, and handling errors in this codebase

## Getting Started

```bash
git clone https://github.com/jt828/go-grpc-template.git
cd go-grpc-template
```

Run the server:

```bash
HOSTNAME="my-service-1" DATABASE_DSN="postgres://user:password@localhost:5432/dbname?sslmode=disable&search_path=main" go run cmd/server/main.go
```

## Project Structure

```
go-grpc-template/
├── cmd/                        # Application entry points
│   ├── server/main.go          # gRPC server
│   └── migration/main.go       # Database migration CLI
├── internal/                   # Domain logic (module-scoped)
│   ├── bootstrap/              # Database & snowflake initialization
│   ├── controller/             # gRPC handlers
│   ├── service/                # Business logic
│   └── repository/             # Data access & unit of work
├── pkg/                        # Reusable packages (public API)
│   ├── circuitbreaker/         # Circuit breaker abstraction
│   ├── idempotency/            # Idempotency pattern
│   ├── model/                  # Domain & data entity models
│   ├── observability/          # Logging, metrics, tracing
│   ├── retry/                  # Retry with exponential backoff
│   └── snowflake/              # Distributed ID generation
├── proto/                      # Protocol Buffer definitions & generated code
├── migrations/                 # SQL migration files
└── test/                       # Unit & integration tests
```

## Protobuf Code Generation

### Install Dependencies

```bash
brew install protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Generate Code

```bash
protoc -I=proto/v1 \
  --go_out=proto --go_opt=paths=source_relative \
  --go-grpc_out=proto --go-grpc_opt=paths=source_relative \
  proto/v1/*.proto
```

## Database Migration

### Install CLI

```bash
brew install golang-migrate
```

### Create Migration

```bash
migrate create -ext sql -dir migrations -seq <migration_name>
```

### Run Migration

```bash
DATABASE_DSN="postgres://user:password@localhost:5432/dbname?sslmode=disable&search_path=main" go run cmd/migration/main.go -direction up
```

### Rollback Migration

```bash
DATABASE_DSN="postgres://user:password@localhost:5432/dbname?sslmode=disable&search_path=main" go run cmd/migration/main.go -direction down -steps 1
```

## Testing

### Unit Tests

```bash
go test ./test/unit/ -v
```

### Integration Tests

Requires Docker running.

```bash
go test ./test/integration/ -v -timeout 120s
```

### All Tests

```bash
go test ./... -v -timeout 120s
```
