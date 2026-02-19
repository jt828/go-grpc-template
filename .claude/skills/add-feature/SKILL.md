---
name: add-feature
description: Add a new entity/feature end-to-end in this project. Use when asked to implement a new domain object, e.g. "add a Product feature" or "implement orders".
---

## Checklist — follow in this order

1. [ ] Model (`pkg/model/<entity>.go`)
2. [ ] Migration (`migrations/`)
3. [ ] Repository (`internal/repository/<entity>_repository.go`)
4. [ ] Unit of Work — add to interface + impl (`internal/repository/unit_of_work.go`)
5. [ ] Service (`internal/service/<entity>_service.go`)
6. [ ] Controller (`internal/controller/<entity>_controller.go`)
7. [ ] Wire up in `cmd/server/main.go`
8. [ ] Request type constant if idempotency is needed (`internal/constant/request_type.go`)

---

## Step 1 — Model (`pkg/model/<entity>.go`)

```go
package model

import "time"

type FooDataEntity struct {
    Id        int64     `gorm:"column:id"`
    // ... fields
    CreatedAt time.Time `gorm:"column:created_at"`
}

func (e *FooDataEntity) TableName() string { return "main.foos" }

func (e *FooDataEntity) ToDomain() Foo { return Foo(*e) }

type Foo struct {
    Id        int64
    // ... same fields, no gorm tags
    CreatedAt time.Time
}
```

- Domain model and data entity are always separate structs.
- `TableName()` always prefixes with `main.`.
- `ToDomain()` is a simple cast when the structs are identical: `return Foo(*e)`.
- Use `github.com/shopspring/decimal` (`decimal.Decimal`) for monetary/token amounts.

---

## Step 2 — Migration

```bash
make create-migration
# enter: create_foos_table
```

Edit the generated files:

```sql
-- migrations/NNNN_create_foos_table.up.sql
CREATE TABLE IF NOT EXISTS main.foos (
    id         BIGINT PRIMARY KEY,
    ...
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- migrations/NNNN_create_foos_table.down.sql
DROP TABLE IF EXISTS main.foos;
```

Also add the same `CREATE TABLE` to `test/integration/testdata/init_schema.sql` so integration tests pick it up.

---

## Step 3 — Repository (`internal/repository/foo_repository.go`)

```go
package repository

import (
    "context"
    "errors"

    "github.com/jt828/go-grpc-template/pkg/circuitbreaker"
    "github.com/jt828/go-grpc-template/pkg/model"
    "github.com/jt828/go-grpc-template/pkg/retry"
    "gorm.io/gorm"
)

type FooRepository interface {
    Get(ctx context.Context, id int64) (*model.Foo, error)
    Insert(ctx context.Context, foo *model.Foo) error
}

type FooRepositoryImpl struct {
    db              *gorm.DB
    cb              circuitbreaker.CircuitBreaker
    retry           retry.Retry
    notFoundAsError bool
}

func NewFooRepository(db *gorm.DB, cb circuitbreaker.CircuitBreaker, retry retry.Retry, notFoundAsError bool) FooRepository {
    return &FooRepositoryImpl{db: db, cb: cb, retry: retry, notFoundAsError: notFoundAsError}
}
```

**Get pattern (single record via `First`):**
```go
func (r *FooRepositoryImpl) Get(ctx context.Context, id int64) (*model.Foo, error) {
    result, err := r.cb.Execute(func() (any, error) {
        var foo *model.Foo
        err := r.retry.Execute(ctx, func() error {
            var entity model.FooDataEntity
            if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
                if !r.notFoundAsError && errors.Is(err, gorm.ErrRecordNotFound) {
                    return nil
                }
                return err
            }
            f := entity.ToDomain()
            foo = &f
            return nil
        })
        return foo, err
    })
    if err != nil {
        return nil, err
    }
    return result.(*model.Foo), nil
}
```

**Get pattern (list with filters — use `Find`):**
```go
func (r *FooRepositoryImpl) Get(ctx context.Context, query GetFooQuery) ([]*model.Foo, error) {
    result, err := r.cb.Execute(func() (any, error) {
        var foos []*model.Foo
        err := r.retry.Execute(ctx, func() error {
            var entities []model.FooDataEntity
            db := r.db.WithContext(ctx)
            if query.UserIdEq != 0 {
                db = db.Where("user_id = ?", query.UserIdEq)
            }
            if err := db.Find(&entities).Error; err != nil {
                return err
            }
            foos = make([]*model.Foo, len(entities))
            for i := range entities {
                f := entities[i].ToDomain()
                foos[i] = &f
            }
            return nil
        })
        return foos, err
    })
    if err != nil {
        return nil, err
    }
    return result.([]*model.Foo), nil
}
```

**Insert pattern:**
```go
func (r *FooRepositoryImpl) Insert(ctx context.Context, foo *model.Foo) error {
    _, err := r.cb.Execute(func() (any, error) {
        err := r.retry.Execute(ctx, func() error {
            entity := model.FooDataEntity{Id: foo.Id, CreatedAt: foo.CreatedAt}
            return r.db.WithContext(ctx).Create(&entity).Error
        })
        return nil, err
    })
    return err
}
```

---

## Step 4 — Unit of Work (`internal/repository/unit_of_work.go`)

Add to the `UnitOfWork` interface:
```go
FooRepository() FooRepository
```

Add field + `sync.Once` to `transactionDbUnitOfWork`:
```go
fooRepository     FooRepository
fooRepositoryOnce sync.Once
```

Add the lazy-init method:
```go
func (u *transactionDbUnitOfWork) FooRepository() FooRepository {
    u.fooRepositoryOnce.Do(func() {
        u.fooRepository = NewFooRepository(u.tx, u.cb, u.retry, false)
    })
    return u.fooRepository
}
```

---

## Step 5 — Service (`internal/service/foo_service.go`)

**Without idempotency:**
```go
func (s *fooService) CreateFoo(ctx context.Context, foo *model.Foo) (*model.Foo, error) {
    uow, err := s.uowFactory.New()
    if err != nil {
        return nil, err
    }

    foo.Id = s.snowflake.Generate()
    foo.CreatedAt = time.Now().UTC()

    if err := uow.FooRepository().Insert(ctx, foo); err != nil {
        _ = uow.Abort(ctx)
        return nil, err
    }

    created, err := uow.FooRepository().Get(ctx, foo.Id)
    if err != nil {
        _ = uow.Abort(ctx)
        return nil, err
    }

    if err := uow.Commit(ctx); err != nil {
        return nil, err
    }
    return created, nil
}
```

**With idempotency** (add `idempotencyId int64` param, inject `idempotency.Idempotency`):
```go
result, err := s.idempotency.Execute(
    ctx,
    uow.IdempotencyRecordRepository(),
    idempotencyId,
    constant.RequestTypeCreateFoo,   // add to internal/constant/request_type.go
    foo.Id,
    func() any { return &model.Foo{} },
    func() (any, error) {
        if err := uow.FooRepository().Insert(ctx, foo); err != nil {
            return nil, err
        }
        return uow.FooRepository().Get(ctx, foo.Id)
    },
)
if err != nil {
    _ = uow.Abort(ctx)
    return nil, err
}
if err := uow.Commit(ctx); err != nil {
    return nil, err
}
return result.(*model.Foo), nil
```

---

## Step 6 — Controller (`internal/controller/foo_controller.go`)

```go
package controller

import (
    "context"

    "github.com/jt828/go-grpc-template/internal/service"
    "github.com/jt828/go-grpc-template/pkg/model"
    v1 "github.com/jt828/go-grpc-template/proto"
    "google.golang.org/protobuf/types/known/timestamppb"
)

type FooController struct {
    v1.UnimplementedFooServiceServer
    fooService service.FooService
}

func NewFooController(fooService service.FooService) *FooController {
    return &FooController{fooService: fooService}
}

func (ctrl *FooController) CreateFoo(ctx context.Context, req *v1.CreateFooRequest) (*v1.CreateFooResponse, error) {
    foo := &model.Foo{/* map from req */}
    created, err := ctrl.fooService.CreateFoo(ctx, req.IdempotencyId, foo)
    if err != nil {
        return nil, err
    }
    return &v1.CreateFooResponse{
        Id:        created.Id,
        CreatedAt: timestamppb.New(created.CreatedAt),
    }, nil
}
```

- Embed `v1.UnimplementedFooServiceServer` for forward compatibility.
- Convert `time.Time` → proto with `timestamppb.New(t)`.
- Never return a domain model directly; always map to proto response.

---

## Step 7 — Wire up in `cmd/server/main.go`

```go
fooSvc  := service.NewFooService(dbs.UnitOfWorkFactory, idem, idGen)
fooCtrl := controller.NewFooController(fooSvc)
v1.RegisterFooServiceServer(server, fooCtrl)
```

Insert after the existing service/controller wiring, before `grpcMetrics.InitializeMetrics(server)`.

---

## Step 8 — Request type constant (if using idempotency)

`internal/constant/request_type.go`:
```go
const (
    RequestTypeCreateUser RequestType = "create_user"
    RequestTypeCreateFoo  RequestType = "create_foo"   // add this
)
```