---
name: run-tests
description: Run tests in this project. Use when asked to run, execute, or check tests — whether unit, integration, or a specific test by name.
allowed-tools: Bash
---

## Test commands

### Run all unit tests
```bash
make test-unit
# equivalent: go test ./test/unit/ -v
```

### Run all integration tests
```bash
make test-integration
# equivalent: go test ./test/integration/ -v -timeout 120s
```

Integration tests spin up a real Postgres container via testcontainers. They are slower (~1–2s per test function) and require Docker to be running.

### Run a specific test by name
```bash
# Unit
go test ./test/unit/ -v -run TestUserRepository_Get

# Integration
go test ./test/integration/ -v -timeout 120s -run TestLedgerRepository_Get_Integration
```

Use `-run` with a regex; it matches against the full test name including subtests:
```bash
# Run all ledger tests
go test ./test/integration/ -v -timeout 120s -run TestLedger

# Run a specific subtest
go test ./test/integration/ -v -timeout 120s -run "TestLedgerRepository_Get_Integration/filter_by_UserIdEq"
```

### Run all tests
```bash
go test ./... -timeout 120s
```

## What to check after running

- All subtests should be `--- PASS`
- For unit tests: `mock.ExpectationsWereMet()` failures mean the SQL query didn't match — check the exact query GORM generates (logged in red in the output) and update the `ExpectQuery` string
- For integration tests: container startup failures usually mean Docker isn't running