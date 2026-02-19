package bootstrap

import (
	"errors"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jt828/go-grpc-template/pkg/circuitbreaker"
	cbImpl "github.com/jt828/go-grpc-template/pkg/circuitbreaker/implementation"
	"github.com/jt828/go-grpc-template/pkg/observability"
	obsImpl "github.com/jt828/go-grpc-template/pkg/observability/implementation"
	"github.com/jt828/go-grpc-template/pkg/retry"
	retryImpl "github.com/jt828/go-grpc-template/pkg/retry/implementation"
	"github.com/jt828/go-grpc-template/internal/repository"
	"github.com/sony/gobreaker/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	DB             *gorm.DB
	CircuitBreaker circuitbreaker.CircuitBreaker
	UnitOfWorkFactory repository.UnitOfWorkFactory
}

func InitializeDatabase(dsn string, meter observability.Meter) (*Database, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.Use(obsImpl.NewGormMetricsPlugin(meter)); err != nil {
		return nil, err
	}

	cb := cbImpl.NewCircuitBreaker(gobreaker.Settings{
		Name: "postgresql",
	})

	retry := retryImpl.NewRetry(3, retry.WithInterval(100*time.Millisecond), retry.WithRetryable(func(err error) bool {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "40001": // serialization_failure
				return true
			case "40P01": // deadlock_detected
				return true
			case "08006": // connection_failure
				return true
			case "08001": // sqlclient_unable_to_establish_sqlconnection
				return true
			case "08004": // sqlserver_rejected_establishment_of_sqlconnection
				return true
			}
		}

		var netErr *net.OpError
		if errors.As(err, &netErr) {
			return true
		}

		return false
	}))
	uowFactory := repository.NewTransactionDbUnitOfWorkFactory(db, cb, retry)

	return &Database{
		DB:                db,
		CircuitBreaker:    cb,
		UnitOfWorkFactory: uowFactory,
	}, nil
}
