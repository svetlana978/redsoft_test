package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"person-service/internal/api/grpc"
	"person-service/internal/repository"
	"person-service/internal/service"
	"person-service/pkg/config"
	"person-service/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()

	lg, err := logger.New(cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer lg.Sync()

	lg.Info("starting person service",
		zap.String("version", "1.0.0"),
		zap.String("grpc_port", cfg.GRPCPort),
	)

	dbConfig := &repository.DBConfig{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
		ConnMaxIdleTime: cfg.DBConnMaxIdleTime,
	}

	repo, err := repository.NewPostgresRepository(cfg.GetDBConnString(), dbConfig, lg)
	if err != nil {
		lg.Fatal("failed to initialize database", zap.Error(err))
	}
	defer repo.Close()

	enrich := service.NewEnrichmentService(
		lg,
		cfg.AgifyURL,
		cfg.GenderizeURL,
		cfg.NationalizeURL,
		cfg.EnrichTimeout,
		cfg.AllowTimeout,
	)

	personSvc := service.NewPersonService(repo, enrich, lg)

	grpcServer := grpc.NewServer(personSvc, lg, cfg.GRPCPort, cfg.GRPCGracefulTimeout)

	go func() {
		if err := grpcServer.Start(); err != nil {
			lg.Fatal("failed to start server person-service", zap.Error(err))
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()
	<-ctx.Done()

	grpcServer.GracefulStop()

	lg.Info("server person-service stopped")
}
