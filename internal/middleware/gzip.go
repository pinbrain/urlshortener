package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/pinbrain/urlshortener/internal/logger"
)

// compressWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки.
type compressWriter struct {
	w        http.ResponseWriter
	zw       *gzip.Writer
	needGzip bool
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:        w,
		zw:       gzip.NewWriter(w),
		needGzip: false,
	}
}

// Header возвращает пары ключ-значения для получения и установки значений заголовков.
func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

// Write переопределяет штатный метод http.ResponseWriter записи ответа.
// Сжимает ответ, если был выставлен признак needGzip = true.
func (c *compressWriter) Write(p []byte) (int, error) {
	if c.needGzip {
		return c.zw.Write(p)
	}
	return c.w.Write(p)
}

// WriteHeader штатный метод http.ResponseWriter записи заголовка ответа.
// Определяет необходимость сжатия ответа и устанавливает соответствующий заголовок ответа.
func (c *compressWriter) WriteHeader(statusCode int) {
	resContentType := c.w.Header().Get("Content-Type")
	c.needGzip = strings.Contains(resContentType, "application/json") || strings.Contains(resContentType, "text/html")
	if c.needGzip {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и досылает все данные из буфера.
func (c *compressWriter) Close() error {
	if c.needGzip {
		return c.zw.Close()
	}
	return nil
}

// compressReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные.
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

// Read читает входящие данные и распаковывает их.
func (c compressReader) Read(p []byte) (int, error) {
	return c.zr.Read(p)
}

// Close закрывает Reader.
func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

// GzipMiddleware создает обработчик поддерживающий сжатие и декомпрессию данных.
func GzipMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			cw := newCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				logger.Log.Errorw("Error in reading compressed request body", "err", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		h.ServeHTTP(ow, r)
	})
}
