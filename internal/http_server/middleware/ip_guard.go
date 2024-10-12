package middleware

import (
	"net"
	"net/http"
)

// IPGuardMiddleware описывает структуру обработчика блокирующего доступ для ip не из доверенной подсети.
type IPGuardMiddleware struct {
	trustedSubnet *net.IPNet
}

// NewIPGuardMiddleware создает обработчик блокирующий доступ для ip не из доверенной подсети.
func NewIPGuardMiddleware(trustedSubnet *net.IPNet) IPGuardMiddleware {
	return IPGuardMiddleware{
		trustedSubnet: trustedSubnet,
	}
}

// GuardByIP проверяет что ip запроса входит в доверенную подсеть.
// В противном случае прерывает обработку запроса и возвращает ошибку Forbidden.
func (ipmw *IPGuardMiddleware) GuardByIP(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ipmw.trustedSubnet == nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		clientIP := r.Header.Get("X-Real-IP")
		if clientIP == "" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		ip := net.ParseIP(clientIP)
		if ip == nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if !ipmw.trustedSubnet.Contains(ip) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, r)
	})
}
