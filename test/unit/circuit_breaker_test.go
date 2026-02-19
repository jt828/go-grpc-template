package unit

import (
	"errors"
	"testing"
	"time"

	"github.com/jt828/go-grpc-template/pkg/circuitbreaker"
	cbImpl "github.com/jt828/go-grpc-template/pkg/circuitbreaker/implementation"
	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreaker_Execute(t *testing.T) {
	t.Run("successful execution returns result", func(t *testing.T) {
		cb := cbImpl.NewCircuitBreaker(gobreaker.Settings{Name: "test"})

		result, err := cb.Execute(func() (any, error) {
			return "hello", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("failed execution returns error", func(t *testing.T) {
		cb := cbImpl.NewCircuitBreaker(gobreaker.Settings{Name: "test"})
		expected := errors.New("operation failed")

		result, err := cb.Execute(func() (any, error) {
			return nil, expected
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, expected)
	})

	t.Run("opens after reaching failure threshold", func(t *testing.T) {
		cb := cbImpl.NewCircuitBreaker(gobreaker.Settings{
			Name: "test",
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 3
			},
		})

		opErr := errors.New("fail")
		for i := 0; i < 3; i++ {
			cb.Execute(func() (any, error) { return nil, opErr })
		}

		assert.Equal(t, circuitbreaker.Open, cb.State())

		result, err := cb.Execute(func() (any, error) {
			t.Fatal("should not be called when circuit is open")
			return nil, nil
		})

		assert.Nil(t, result)
		assert.Error(t, err)
	})
}

func TestCircuitBreaker_State(t *testing.T) {
	t.Run("initial state is closed", func(t *testing.T) {
		cb := cbImpl.NewCircuitBreaker(gobreaker.Settings{Name: "test"})
		assert.Equal(t, circuitbreaker.Closed, cb.State())
	})

	t.Run("state is open after failures trip breaker", func(t *testing.T) {
		cb := cbImpl.NewCircuitBreaker(gobreaker.Settings{
			Name: "test",
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 1
			},
		})

		cb.Execute(func() (any, error) { return nil, errors.New("fail") })
		assert.Equal(t, circuitbreaker.Open, cb.State())
	})

	t.Run("state transitions to half-open after timeout", func(t *testing.T) {
		cb := cbImpl.NewCircuitBreaker(gobreaker.Settings{
			Name: "test",
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 1
			},
			Timeout: time.Millisecond, // very short timeout to transition to half-open
		})

		cb.Execute(func() (any, error) { return nil, errors.New("fail") })
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, circuitbreaker.HalfOpen, cb.State())
	})
}
