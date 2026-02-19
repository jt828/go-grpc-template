package unit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jt828/go-grpc-template/internal/constant"
	"github.com/jt828/go-grpc-template/pkg/idempotency"
	"github.com/jt828/go-grpc-template/pkg/idempotency/implementation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testResult struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type mockRecordRepository struct {
	getFunc    func(ctx context.Context, id int64) (*idempotency.Record, error)
	insertFunc func(ctx context.Context, record *idempotency.Record) error
}

func (m *mockRecordRepository) Get(ctx context.Context, id int64) (*idempotency.Record, error) {
	return m.getFunc(ctx, id)
}

func (m *mockRecordRepository) Insert(ctx context.Context, record *idempotency.Record) error {
	return m.insertFunc(ctx, record)
}

func TestIdempotencyExecute(t *testing.T) {
	ctx := context.Background()
	idempotencyId := int64(100)
	referenceId := int64(200)
	requestType := constant.RequestTypeCreateUser

	newResult := func() any {
		return &testResult{}
	}

	t.Run("cache miss executes function and inserts record", func(t *testing.T) {
		expected := &testResult{Name: "alice", Value: 42}
		var insertedRecord *idempotency.Record

		repo := &mockRecordRepository{
			getFunc: func(ctx context.Context, id int64) (*idempotency.Record, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, record *idempotency.Record) error {
				insertedRecord = record
				return nil
			},
		}

		idem := implementation.NewIdempotency()
		result, err := idem.Execute(ctx, repo, idempotencyId, requestType, referenceId, newResult, func() (any, error) {
			return expected, nil
		})

		require.NoError(t, err)
		assert.Equal(t, expected, result)

		require.NotNil(t, insertedRecord)
		assert.Equal(t, idempotencyId, insertedRecord.Id)
		assert.Equal(t, string(requestType), insertedRecord.RequestType)
		assert.Equal(t, referenceId, insertedRecord.ReferenceId)
		assert.False(t, insertedRecord.CreatedAt.IsZero())

		var stored testResult
		require.NoError(t, json.Unmarshal([]byte(insertedRecord.ResponseData), &stored))
		assert.Equal(t, *expected, stored)
	})

	t.Run("cache hit returns deserialized result without executing function", func(t *testing.T) {
		cached := &testResult{Name: "bob", Value: 99}
		data, _ := json.Marshal(cached)

		repo := &mockRecordRepository{
			getFunc: func(ctx context.Context, id int64) (*idempotency.Record, error) {
				return &idempotency.Record{
					Id:           idempotencyId,
					RequestType:  string(requestType),
					ReferenceId:  referenceId,
					ResponseData: string(data),
				}, nil
			},
			insertFunc: func(ctx context.Context, record *idempotency.Record) error {
				t.Fatal("insert should not be called on cache hit")
				return nil
			},
		}

		fnCalled := false
		idem := implementation.NewIdempotency()
		result, err := idem.Execute(ctx, repo, idempotencyId, requestType, referenceId, newResult, func() (any, error) {
			fnCalled = true
			return nil, nil
		})

		require.NoError(t, err)
		assert.False(t, fnCalled)
		assert.Equal(t, cached, result)
	})

	t.Run("repo Get error is propagated", func(t *testing.T) {
		repoErr := errors.New("database connection failed")

		repo := &mockRecordRepository{
			getFunc: func(ctx context.Context, id int64) (*idempotency.Record, error) {
				return nil, repoErr
			},
		}

		idem := implementation.NewIdempotency()
		result, err := idem.Execute(ctx, repo, idempotencyId, requestType, referenceId, newResult, func() (any, error) {
			t.Fatal("fn should not be called when Get fails")
			return nil, nil
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, repoErr)
	})

	t.Run("fn error is propagated without inserting record", func(t *testing.T) {
		fnErr := errors.New("business logic failed")

		repo := &mockRecordRepository{
			getFunc: func(ctx context.Context, id int64) (*idempotency.Record, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, record *idempotency.Record) error {
				t.Fatal("insert should not be called when fn fails")
				return nil
			},
		}

		idem := implementation.NewIdempotency()
		result, err := idem.Execute(ctx, repo, idempotencyId, requestType, referenceId, newResult, func() (any, error) {
			return nil, fnErr
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, fnErr)
	})

	t.Run("invalid cached JSON returns unmarshal error", func(t *testing.T) {
		repo := &mockRecordRepository{
			getFunc: func(ctx context.Context, id int64) (*idempotency.Record, error) {
				return &idempotency.Record{
					Id:           idempotencyId,
					ResponseData: "not valid json{{{",
				}, nil
			},
		}

		idem := implementation.NewIdempotency()
		result, err := idem.Execute(ctx, repo, idempotencyId, requestType, referenceId, newResult, func() (any, error) {
			t.Fatal("fn should not be called when cache hit")
			return nil, nil
		})

		assert.Nil(t, result)
		assert.Error(t, err)
		var syntaxErr *json.SyntaxError
		assert.ErrorAs(t, err, &syntaxErr)
	})

	t.Run("marshal error is propagated when result is not serializable", func(t *testing.T) {
		repo := &mockRecordRepository{
			getFunc: func(ctx context.Context, id int64) (*idempotency.Record, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, record *idempotency.Record) error {
				t.Fatal("insert should not be called when marshal fails")
				return nil
			},
		}

		idem := implementation.NewIdempotency()
		result, err := idem.Execute(ctx, repo, idempotencyId, requestType, referenceId, newResult, func() (any, error) {
			return func() {}, nil // functions are not JSON-serializable
		})

		assert.Nil(t, result)
		assert.Error(t, err)
		var marshalErr *json.UnsupportedTypeError
		assert.ErrorAs(t, err, &marshalErr)
	})

	t.Run("repo Insert error is propagated", func(t *testing.T) {
		insertErr := errors.New("insert failed")

		repo := &mockRecordRepository{
			getFunc: func(ctx context.Context, id int64) (*idempotency.Record, error) {
				return nil, nil
			},
			insertFunc: func(ctx context.Context, record *idempotency.Record) error {
				return insertErr
			},
		}

		idem := implementation.NewIdempotency()
		result, err := idem.Execute(ctx, repo, idempotencyId, requestType, referenceId, newResult, func() (any, error) {
			return &testResult{Name: "test", Value: 1}, nil
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, insertErr)
	})
}
