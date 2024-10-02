package interceptors

import (
	"context"
	"net"
	"testing"

	pb "github.com/pinbrain/urlshortener/internal/grpc_server/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGuardByIP(t *testing.T) {
	handler := func(_ context.Context, req any) (any, error) {
		return req, nil
	}

	tests := []struct {
		name    string
		cidr    string
		method  string
		meta    map[string]string
		wantErr bool
		errCode codes.Code
	}{
		{
			name:    "Успешный запрос защищенного метода",
			cidr:    "192.168.0.0/24",
			method:  pb.URLShortener_GetStats_FullMethodName,
			meta:    map[string]string{ipMetaKey: "192.168.0.1"},
			wantErr: false,
		},
		{
			name:    "IP не из разрешенной подсети",
			cidr:    "192.168.0.0/24",
			method:  pb.URLShortener_GetStats_FullMethodName,
			meta:    map[string]string{ipMetaKey: "192.168.1.1"},
			wantErr: true,
			errCode: codes.PermissionDenied,
		},
		{
			name:    "Нет meta key с ip",
			cidr:    "192.168.0.0/24",
			method:  pb.URLShortener_GetStats_FullMethodName,
			wantErr: true,
			errCode: codes.PermissionDenied,
		},
		{
			name:    "Разрешенная подсеть не задана",
			method:  pb.URLShortener_GetStats_FullMethodName,
			wantErr: true,
			errCode: codes.PermissionDenied,
		},
		{
			name:    "Незащищенный метод",
			method:  pb.URLShortener_GetURL_FullMethodName,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var trustedSubnet *net.IPNet
			var err error
			if tt.cidr != "" {
				_, trustedSubnet, err = net.ParseCIDR(tt.cidr)
				require.NoError(t, err)
			}
			interceptor := NewIPGuardInterceptor(trustedSubnet)
			md := metadata.New(tt.meta)
			ctx := metadata.NewIncomingContext(context.Background(), md)
			info := &grpc.UnaryServerInfo{FullMethod: tt.method}
			_, err = interceptor.GuardByIP(ctx, nil, info, handler)
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				code, _ := status.FromError(err)
				assert.Equal(t, tt.errCode, code.Code())
			}
		})
	}
}
