package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jt828/go-grpc-template/pkg/retry"
	retryImpl "github.com/jt828/go-grpc-template/pkg/retry/implementation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetry_Execute(t *testing.T) {
	t.Run("succeeds on first attempt", func(t *testing.T) {
		r := retryImpl.NewRetry(3, retry.WithInterval(time.Millisecond))
		callCount := 0

		err := r.Execute(context.Background(), func() error {
			callCount++
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("succeeds after retries", func(t *testing.T) {
		r := retryImpl.NewRetry(3,
			retry.WithInterval(time.Millisecond),
			retry.WithRetryable(func(err error) bool { return true }),
		)
		callCount := 0

		err := r.Execute(context.Background(), func() error {
			callCount++
			if callCount < 3 {
				return errors.New("transient error")
			}
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, 3, callCount)
	})

	t.Run("returns error after max retries exhausted", func(t *testing.T) {
		r := retryImpl.NewRetry(2,
			retry.WithInterval(time.Millisecond),
			retry.WithRetryable(func(err error) bool { return true }),
		)
		callCount := 0
		persistentErr := errors.New("persistent error")

		err := r.Execute(context.Background(), func() error {
			callCount++
			return persistentErr
		})

		assert.Error(t, err)
		assert.ErrorContains(t, err, "persistent error")
		// initial attempt + 2 retries = 3 calls
		assert.Equal(t, 3, callCount)
	})

	t.Run("non-retryable error fails immediately", func(t *testing.T) {
		r := retryImpl.NewRetry(3,
			retry.WithInterval(time.Millisecond),
			retry.WithRetryable(func(err error) bool { return false }),
		)
		callCount := 0

		err := r.Execute(context.Background(), func() error {
			callCount++
			return errors.New("fatal error")
		})

		assert.Error(t, err)
		assert.ErrorContains(t, err, "fatal error")
		assert.Equal(t, 1, callCount)
	})

	t.Run("selectively retries based on error type", func(t *testing.T) {
		retryableErr := errors.New("retryable")
		fatalErr := errors.New("fatal")

		r := retryImpl.NewRetry(5,
			retry.WithInterval(time.Millisecond),
			retry.WithRetryable(func(err error) bool {
				return errors.Is(err, retryableErr)
			}),
		)
		callCount := 0

		err := r.Execute(context.Background(), func() error {
			callCount++
			if callCount <= 2 {
				return retryableErr
			}
			return fatalErr
		})

		assert.ErrorContains(t, err, "fatal")
		assert.Equal(t, 3, callCount)
	})

	t.Run("nil retryableFn treats all errors as retryable", func(t *testing.T) {
		r := retryImpl.NewRetry(2, retry.WithInterval(time.Millisecond))
		callCount := 0

		err := r.Execute(context.Background(), func() error {
			callCount++
			if callCount < 3 {
				return errors.New("error")
			}
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, 3, callCount)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		r := retryImpl.NewRetry(100,
			retry.WithInterval(time.Second),
			retry.WithRetryable(func(err error) bool { return true }),
		)

		ctx, cancel := context.WithCancel(context.Background())
		callCount := 0

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		err := r.Execute(ctx, func() error {
			callCount++
			return errors.New("keep failing")
		})

		assert.Error(t, err)
		assert.LessOrEqual(t, callCount, 3)
	})
}
