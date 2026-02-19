package circuitbreaker

type State int

const (
	Closed   State = iota
	HalfOpen
	Open
)

type CircuitBreaker interface {
	Execute(fn func() (any, error)) (any, error)
	State() State
}
