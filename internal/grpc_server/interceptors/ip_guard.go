package interceptors

import (
	"context"
	"net"

	pb "github.com/pinbrain/urlshortener/internal/grpc_server/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Ключ с реальным ip адресом запроса.
const ipMetaKey = "X-Real-IP"

// Методы с ограниченным доступом по ip.
var ipProtectedMethods = map[string]bool{
	pb.URLShortener_GetStats_FullMethodName: true,
}

// IPGuardInterceptor описывает структуру перехватчика блокирующего доступ для ip не из доверенной подсети.
type IPGuardInterceptor struct {
	trustedSubnet *net.IPNet
}

// NewIPGuardInterceptor создает перехватчик блокирующий доступ для ip не из доверенной подсети.
func NewIPGuardInterceptor(trustedSubnet *net.IPNet) *IPGuardInterceptor {
	return &IPGuardInterceptor{
		trustedSubnet: trustedSubnet,
	}
}

// GuardByIP проверяет что ip запроса входит в доверенную подсеть.
// В противном случае прерывает обработку запроса и возвращает ошибку Forbidden.
func (i *IPGuardInterceptor) GuardByIP(
	ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (interface{}, error) {
	if !ipProtectedMethods[info.FullMethod] {
		return handler(ctx, req)
	}
	if i.trustedSubnet == nil {
		return nil, status.Error(codes.PermissionDenied, "Forbidden")
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get(ipMetaKey)
		if len(values) > 0 {
			ip := net.ParseIP(values[0])
			if ip != nil && i.trustedSubnet.Contains(ip) {
				return handler(ctx, req)
			}
		}
	}
	return nil, status.Error(codes.PermissionDenied, "Forbidden")
}
