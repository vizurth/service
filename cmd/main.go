package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"net/http"
	"os/signal"
	"serviceLyceum/internal/config"
	"serviceLyceum/internal/service"
	test "serviceLyceum/pkg/api/test/api"
	"serviceLyceum/pkg/logger"
	"serviceLyceum/pkg/postgres"
	"syscall"
	"time"
)

func main() {
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT) // контект который будет опираться на сигналы системы
	defer stop()
	ctx, _ = logger.New(ctx)

	cfg, err := config.New()
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Fatal(ctx, "config.New error", zap.Error(err))
	}

	pool, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Fatal(ctx, "Failed to connect to database", zap.Error(err))
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", cfg.GRPCPort))

	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "failed to listen", zap.Error(err))
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(addLogMiddleware))
	srv := orderservice.NewService()

	test.RegisterOrderServiceServer(server, srv)

	rt := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	err = test.RegisterOrderServiceHandlerFromEndpoint(ctx, rt, "localhost:"+cfg.GRPCPort, opts)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Fatal(ctx, "failed to register handler server", zap.Error(err))
	}
	fmt.Printf("Starting gRPC server on port %s\n", cfg.GRPCPort)
	fmt.Printf("Starting REST server on port %s\n", cfg.RESTPort)

	go func() {
		if err := http.ListenAndServe(fmt.Sprintf("localhost:%s", cfg.RESTPort), rt); err != nil {
			logger.GetLoggerFromCtx(ctx).Fatal(ctx, "failed to start REST server", zap.Error(err))
		}
	}()

	go func() { // блогировка го рутины
		if err := server.Serve(lis); err != nil {
			logger.GetLoggerFromCtx(ctx).Info(ctx, "failed to start GRPC server", zap.Error(err))
		}
	}()

	select {
	case <-ctx.Done(): // реализация gracefull shot down
		server.Stop()
		pool.Close()
		logger.GetLoggerFromCtx(ctx).Fatal(ctx, "Server stopped")
	}

}

func addLogMiddleware(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	guid := uuid.New().String()
	ctx = context.WithValue(ctx, logger.RequestID, guid)
	ctx, _ = logger.New(ctx)
	logger.GetLoggerFromCtx(ctx).Info(ctx, "gRPC interception", zap.String("method", info.FullMethod), zap.Time("request time", time.Now()))
	res, err := handler(ctx, req)
	return res, err
}
