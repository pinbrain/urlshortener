package middleware

import (
	"net/http"
	"time"

	"github.com/pinbrain/urlshortener/internal/logger"
)

type (
	// responseData описывает структуру сохраняемых данных ответа для их последующего логирования.
	responseData struct {
		status int
		size   int
	}

	// loggingResponseWriter описывает структуру расширенного http.ResponseWriter.
	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

// Write переопределяет штатный метод http.ResponseWriter записи ответа.
// Добавляет сохранение данных о размере тела ответа в структуру.
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

// WriteHeader штатный метод http.ResponseWriter записи заголовка ответа.
// Добавляет сохранение кода ответа в структуру.
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

// HTTPRequestLogger создает обработчик логирующий входящие запросы.
// Логируются данные запроса и ответа, а так же продолжительность обработки запроса.
func HTTPRequestLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}

		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		logger.Log.Infow(
			"HTTP request",
			"uri", r.RequestURI,
			"method", r.Method,
			"status", responseData.status,
			"duration", duration,
			"responseSize", responseData.size,
		)
	})
}
