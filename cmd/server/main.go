package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/jt828/go-grpc-template/internal/bootstrap"
	"github.com/jt828/go-grpc-template/internal/controller"
	"github.com/jt828/go-grpc-template/internal/interceptor"
	"github.com/jt828/go-grpc-template/internal/service"
	idempotencyImpl "github.com/jt828/go-grpc-template/pkg/idempotency/implementation"
	"github.com/jt828/go-grpc-template/pkg/observability"
	"github.com/jt828/go-grpc-template/pkg/observability/implementation"
	v1 "github.com/jt828/go-grpc-template/proto"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := implementation.Config{ServiceName: "go-grpc-template"}
	obs, err := implementation.NewObservability(cfg)
	if err != nil {
		panic(err)
	}
	log := obs.Logger()
	reg := implementation.PromRegistry(obs.Meter())
	if reg == nil {
		log.Fatal("prometheus registry not available")
	}

	grpcMetrics := grpc_prometheus.NewServerMetrics()
	reg.MustRegister(grpcMetrics)

	if err := obs.Start(ctx); err != nil {
		log.Error("failed to start observability", observability.Err(err))
	}

	idGen, err := bootstrap.InitializeSnowflake()
	if err != nil {
		log.Fatal("failed to initialize snowflake", observability.Err(err))
	}
	dsn := os.Getenv("DATABASE_DSN")
	dbs, err := bootstrap.InitializeDatabase(dsn, obs.Meter())
	if err != nil {
		log.Fatal("failed to initialize database", observability.Err(err))
	}

	idem := idempotencyImpl.NewIdempotency()
	userSvc := service.NewUserService(dbs.UnitOfWorkFactory, idem, idGen)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sig
		log.Info("Shutting down server...")
		cancel() // cancel root context
	}()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Info("failed to listen: %v", observability.Err(err))
	}

	server := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			grpcMetrics.UnaryServerInterceptor(),
			interceptor.ErrorInterceptor(log),
		),
		grpc.StreamInterceptor(grpcMetrics.StreamServerInterceptor()),
	)

	userCtrl := controller.NewUserController(userSvc)

	v1.RegisterUserServiceServer(server, userCtrl)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	if sqlDB, err := dbs.DB.DB(); err != nil {
		log.Error("failed to get sql db for health check", observability.Err(err))
	} else if err := sqlDB.PingContext(ctx); err != nil {
		log.Error("database ping failed, server marked as not serving", observability.Err(err))
	} else {
		healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	}

	grpc_health_v1.RegisterHealthServer(server, healthServer)

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sqlDB, err := dbs.DB.DB()
				if err != nil || sqlDB.PingContext(ctx) != nil {
					healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
				} else {
					healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
				}
			}
		}
	}()

	grpcMetrics.InitializeMetrics(server)

	go func() {
		log.Info("gRPC server running on :50051")
		if err := server.Serve(lis); err != nil {
			log.Fatal("failed to serve: %v", observability.Err(err))
		}
	}()

	<-ctx.Done()
	log.Info("Graceful stopping gRPC server...")
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	server.GracefulStop()
	log.Info("gRPC server stopped")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := obs.Close(shutdownCtx); err != nil {
		log.Error("failed to close observability", observability.Err(err))
	}
}
