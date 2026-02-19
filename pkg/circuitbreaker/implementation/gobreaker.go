package implementation

import (
	"github.com/jt828/go-grpc-template/pkg/circuitbreaker"
	"github.com/sony/gobreaker/v2"
)

type gobreakerCircuitBreaker struct {
	cb *gobreaker.CircuitBreaker[any]
}

func NewCircuitBreaker(settings gobreaker.Settings) circuitbreaker.CircuitBreaker {
	return &gobreakerCircuitBreaker{
		cb: gobreaker.NewCircuitBreaker[any](settings),
	}
}

func (g *gobreakerCircuitBreaker) Execute(fn func() (any, error)) (any, error) {
	return g.cb.Execute(fn)
}

func (g *gobreakerCircuitBreaker) State() circuitbreaker.State {
	switch g.cb.State() {
	case gobreaker.StateClosed:
		return circuitbreaker.Closed
	case gobreaker.StateHalfOpen:
		return circuitbreaker.HalfOpen
	case gobreaker.StateOpen:
		return circuitbreaker.Open
	default:
		return circuitbreaker.Closed
	}
}
