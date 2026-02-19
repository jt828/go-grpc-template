package retry

import (
	"context"
	"time"
)

type Retry interface {
	Execute(ctx context.Context, fn func() error) error
}

type Config struct {
	RetryableFn func(err error) bool
	Interval    time.Duration
}

type Option func(*Config)

func WithRetryable(fn func(err error) bool) Option {
	return func(c *Config) {
		c.RetryableFn = fn
	}
}

func WithInterval(d time.Duration) Option {
	return func(c *Config) {
		c.Interval = d
	}
}

func ApplyOptions(opts ...Option) *Config {
	c := &Config{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
