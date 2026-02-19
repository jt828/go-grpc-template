---
name: write-tests
description: Write unit or integration tests for this project. Use when asked to implement tests for a repository, service, or any Go package in go-grpc-template.
---

## Project overview

Module: `github.com/jt828/go-grpc-template`
Language: Go 1.25
Test framework: `github.com/stretchr/testify`

---

## Unit tests (`test/unit/`, package `unit`)

### Tools
- `github.com/DATA-DOG/go-sqlmock` for DB queries
- `gorm.io/driver/postgres` + sqlmock for GORM

### Shared helpers already defined in `test/unit/ledger_repository_test.go`

```go
// No-op circuit breaker and retry — reuse across all unit test files
type passthroughCB struct{}
func (p *passthroughCB) Execute(fn func() (any, error)) (any, error) { return fn() }
func (p *passthroughCB) State() circuitbreaker.State                 { return circuitbreaker.Closed }

type passthroughRetry struct{}
func (p *passthroughRetry) Execute(ctx context.Context, fn func() error) error { return fn() }

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock)  // postgres dialect over sqlmock
```

### SQL query patterns

GORM generates different table references depending on the method:
- `Find` / `Where` → full schema: `"main"."ledgers"`, `"main"."users"`
- `First` → short table name in WHERE/ORDER BY: `"users"."id"`, NOT `"main"."users"."id"`

Example for a `First` call:
```go
mock.ExpectQuery(regexp.QuoteMeta(
    `SELECT * FROM "main"."users" WHERE "users"."id" = $1 ORDER BY "users"."id" LIMIT $2`,
)).WithArgs(int64(1), 1).WillReturnRows(...)
```

Example for a `Find` call:
```go
mock.ExpectQuery(regexp.QuoteMeta(
    `SELECT * FROM "main"."ledgers" WHERE user_id = $1`,
)).WithArgs(int64(10)).WillReturnRows(...)
```

### Standard test cases for a repository

**Get (by ID / filters)**
- happy path: returns domain object with all fields preserved
- not found returns nil (when `notFoundAsError: false`)
- not found returns error (when `notFoundAsError: true`) — only if the repo supports this flag
- DB error is propagated
- entity-to-domain conversion preserves all fields

**Insert**
- successful insert (use `ExpectBegin` / `ExpectCommit` around the query)
- DB error is propagated (use `ExpectBegin` / `ExpectRollback`)

---

## Integration tests (`test/integration/`, package `integration`)

### Tools
- `github.com/testcontainers/testcontainers-go/modules/postgres` — spins up `postgres:16-alpine`
- `testdata/init_schema.sql` — initialised on container start; add tables there as needed

### Shared helpers already defined in `test/integration/user_repository_test.go`

```go
type testDB struct {
    db        *gorm.DB
    container *tcpostgres.PostgresContainer
}

// Starts a fresh Postgres container; registers t.Cleanup to terminate it.
func setupTestDB(t *testing.T) *testDB

// Seed helpers
func seedUser(t *testing.T, db *gorm.DB, user *model.UserDataEntity)
func seedLedger(t *testing.T, db *gorm.DB, ledger *model.LedgerDataEntity)
```

### Standard wiring for repository integration tests

```go
cb := cbImpl.NewCircuitBreaker(gobreaker.Settings{Name: "test"})
r  := retryImpl.NewRetry(3,
    retry.WithInterval(100*time.Millisecond),
    retry.WithRetryable(func(err error) bool { return false }),
)
repo := repository.NewXxxRepository(tdb.db, cb, r, false)
```

### Standard wiring for service integration tests

```go
func setupUserService(t *testing.T, db *gorm.DB) service.UserService {
    cb  := cbImpl.NewCircuitBreaker(gobreaker.Settings{Name: "test"})
    r   := retryImpl.NewRetry(3, retry.WithInterval(100*time.Millisecond),
               retry.WithRetryable(func(err error) bool { return false }))
    uowFactory := repository.NewTransactionDbUnitOfWorkFactory(db, cb, r)
    idem       := idempotencyImpl.NewIdempotency()
    sf, err    := snowflakeImpl.NewSnowflake(1)
    require.NoError(t, err)
    return service.NewUserService(uowFactory, idem, sf)
}
```

### Standard test cases for a repository integration test

- no filters returns all seeded rows
- filter by each individual field (IdEq, UserIdEq, etc.)
- combined filters
- no match returns empty slice (not an error)
- insert happy path — read back and assert all fields
- insert duplicate primary key returns error

### Standard test cases for a service integration test

- happy path: returned object has all fields set (snowflake ID non-zero, timestamps ≈ now)
- idempotency: same key returns original result, second call's input is ignored
- different idempotency keys produce distinct records
- round-trip: create then get confirms record persists

---

## Conventions

- Test function names: `TestXxxRepository_Get`, `TestXxxRepository_Insert`, `TestXxxService_GetXxx_Integration`, `TestXxxService_CreateXxx_Integration`
- Sub-test names: plain English describing the scenario (e.g. `"filter by UserIdEq"`, `"duplicate id returns error"`)
- Use `require.NoError` when an error would make subsequent assertions meaningless; use `assert` otherwise
- Truncate times to second precision: `time.Now().Truncate(time.Second)` before seeding
- Use `decimal.NewFromFloat(...)` for `shopspring/decimal` amounts
- Do not add new seed helpers unless they are genuinely reusable; inline `db.Create(...)` for one-off records

---

## Adding a new table

Before writing integration tests for a new model, add its `CREATE TABLE` statement to:
`test/integration/testdata/init_schema.sql`

Match the schema exactly to the corresponding migration file in `migrations/`.