package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"person-service/internal/service"
	pb "person-service/proto"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer      *grpc.Server
	personSvc       *service.PersonService
	logger          *zap.Logger
	port            string
	gracefulTimeout time.Duration
}

func NewServer(personSvc *service.PersonService, logger *zap.Logger, port string, gracefulTimeout time.Duration) *Server {
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			loggingInterceptor(logger),
			recoveryInterceptor(logger),
		),
	)

	return &Server{
		grpcServer:      grpcServer,
		personSvc:       personSvc,
		logger:          logger,
		port:            port,
		gracefulTimeout: gracefulTimeout,
	}
}

func (s *Server) Start() error {
	pb.RegisterPersonServiceServer(s.grpcServer, NewHandlers(s.personSvc, s.logger))
	reflection.Register(s.grpcServer)

	lis, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return err
	}

	s.logger.Info("gRPC server person-service starting", zap.String("port", s.port))

	return s.grpcServer.Serve(lis)
}

func (s *Server) GracefulStop() {
	s.logger.Info("Starting Graceful Stop")

	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("GRPC server gracefully stopped")
	case <-time.After(s.gracefulTimeout):
		s.logger.Info("GRPC autotimeout, without graceful")
	}
}

func loggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.Debug("gRPC request",
			zap.String("method", info.FullMethod),
			zap.Any("request", req))

		resp, err := handler(ctx, req)
		if err != nil {
			logger.Error("gRPC error",
				zap.String("method", info.FullMethod),
				zap.Error(err))
		} else {
			logger.Debug("gRPC response",
				zap.String("method", info.FullMethod))
		}

		return resp, err
	}
}

func recoveryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r))
				err = fmt.Errorf("internal server error")
			}
		}()
		return handler(ctx, req)
	}
}
