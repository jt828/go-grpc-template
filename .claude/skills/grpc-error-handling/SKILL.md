---
name: grpc-error-handling
description: Handle errors in gRPC controllers or the error interceptor. Use when adding validation, handling not-found cases, adding a new sentinel error, or writing tests for error handling.
---

## Architecture

Error handling is split across three layers:

```
Controller  →  returns apperror sentinels (ErrNotFound, ErrInvalidArgument)
Interceptor →  maps sentinels to gRPC status codes, logs unknowns
Service     →  returns raw errors; no knowledge of gRPC or apperrors
```

go-grpc-template is a data access layer — business logic errors (e.g. "email already taken") belong in a higher layer, not here.

---

## Sentinel errors (`pkg/apperror/errors.go`)

```go
var (
    ErrNotFound        = errors.New("not found")
    ErrInvalidArgument = errors.New("invalid argument")
)
```

Wrap with context using `fmt.Errorf`:
```go
fmt.Errorf("user %d: %w", id, apperror.ErrNotFound)
fmt.Errorf("email is required: %w", apperror.ErrInvalidArgument)
```

The interceptor uses `errors.Is()` so wrapping is safe.

---

## Interceptor (`internal/interceptor/error_interceptor.go`)

| Error | gRPC code | Logged? |
|---|---|---|
| `apperror.ErrNotFound` | `codes.NotFound` | No |
| `apperror.ErrInvalidArgument` | `codes.InvalidArgument` | No |
| anything else | `codes.Internal` | Yes — `log.Error` with `"error"` and `"method"` fields |

Unknown errors return `"internal server error"` as the message — internal details never leak to the caller.

Registered in `cmd/server/main.go` via `grpc.ChainUnaryInterceptor`.

---

## Controller conventions

**Validation** — check request fields before calling the service, return wrapped `ErrInvalidArgument`:
```go
if request.Id <= 0 {
    return nil, fmt.Errorf("id must be greater than 0: %w", apperror.ErrInvalidArgument)
}
if request.Email == "" {
    return nil, fmt.Errorf("email is required: %w", apperror.ErrInvalidArgument)
}
```

**Not found** — service returns `nil, nil` when a record doesn't exist; the controller must check:
```go
user, err := ctrl.userService.GetUser(ctx, request.Id)
if err != nil {
    return nil, err   // interceptor handles it
}
if user == nil {
    return nil, fmt.Errorf("user %d: %w", request.Id, apperror.ErrNotFound)
}
```

**Service errors** — pass through as-is; the interceptor maps them:
```go
if err != nil {
    return nil, err
}
```

Controllers never call `status.Error(...)` directly — that's the interceptor's job.

---

## Adding a new sentinel error

Only add when there is a concrete code path that needs it today. Steps:
1. Add the var to `pkg/apperror/errors.go`
2. Add a `case errors.Is(err, apperror.ErrXxx):` to the interceptor switch
3. Use it in the relevant controller

---

## Testing

### Unit test (interceptor in isolation)

```go
// mock logger to capture Error calls
type mockLogger struct {
    errorCalls []struct{ msg string; fields []observability.Field }
}
func (m *mockLogger) Error(msg string, fields ...observability.Field) {
    m.errorCalls = append(m.errorCalls, struct{...}{msg, fields})
}
// ... other methods no-op

fn := interceptor.ErrorInterceptor(log)
_, err := fn(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
    return nil, fmt.Errorf("wrap: %w", apperror.ErrNotFound)
})
assert.Equal(t, codes.NotFound, status.Code(err))
```

Subtests to cover: nil error passthrough, each sentinel (direct + wrapped), unknown error maps to Internal + logs.

### Integration test (interceptor over real gRPC)

Use a real TCP listener + stub `UserServiceServer` that returns a controlled error:
```go
type errorControlledServer struct {
    v1.UnimplementedUserServiceServer
    err error
}
func (s *errorControlledServer) GetUserById(_ context.Context, _ *v1.GetUserByIdRequest) (*v1.GetUserByIdResponse, error) {
    if s.err != nil { return nil, s.err }
    return &v1.GetUserByIdResponse{}, nil
}

func setupInterceptorServer(t *testing.T, svc v1.UserServiceServer) v1.UserServiceClient {
    lis, _ := net.Listen("tcp", "127.0.0.1:0")
    srv := grpc.NewServer(grpc.UnaryInterceptor(interceptor.ErrorInterceptor(&noopLogger{})))
    v1.RegisterUserServiceServer(srv, svc)
    t.Cleanup(func() { srv.GracefulStop() })
    go func() { _ = srv.Serve(lis) }()

    conn, _ := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
    t.Cleanup(func() { _ = conn.Close() })
    return v1.NewUserServiceClient(conn)
}
```

No testcontainers needed — the stub replaces the real service.

A `noopLogger` is shared across the integration package (already defined in `error_interceptor_test.go`).