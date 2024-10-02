package interceptors

import (
	"context"
	"time"

	"github.com/pinbrain/urlshortener/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// LoggerInterceptor логирует входящие запросы.
func LoggerInterceptor(
	ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)
	status, _ := status.FromError(err)

	logger.Log.Infow(
		"gRPC request",
		"method", info.FullMethod,
		"duration", duration,
		"code", status.Code(),
	)
	return resp, err
}
