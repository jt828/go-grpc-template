package implementation

import (
	"context"

	"github.com/jt828/go-grpc-template/pkg/retry"
	goretry "github.com/sethvargo/go-retry"
)

type goRetry struct {
	backoff     goretry.Backoff
	retryableFn func(err error) bool
}

func NewRetry(maxRetries uint64, opts ...retry.Option) retry.Retry {
	cfg := retry.ApplyOptions(opts...)

	backoff := goretry.NewExponential(cfg.Interval)

	return &goRetry{
		backoff:     goretry.WithMaxRetries(maxRetries, backoff),
		retryableFn: cfg.RetryableFn,
	}
}

func (r *goRetry) Execute(ctx context.Context, fn func() error) error {
	return goretry.Do(ctx, r.backoff, func(ctx context.Context) error {
		err := fn()
		if err == nil {
			return nil
		}

		if r.retryableFn != nil && !r.retryableFn(err) {
			return err
		}

		return goretry.RetryableError(err)
	})
}
