package idempotency

import "time"

type Record struct {
	Id           int64
	RequestType  string
	ReferenceId  int64
	ResponseData string
	CreatedAt    time.Time
}
