// Модуль по компрессии
package compress

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// compressWriter позволяет прозрачно для сервера компрессировать получаемые от клиента данные.
type compressWriter struct {
	gin.ResponseWriter
	zw *gzip.Writer
}

// newCompressWriter функция конструктор на запись
func newCompressWriter(w gin.ResponseWriter) *compressWriter {
	return &compressWriter{
		ResponseWriter: w,
		zw:             gzip.NewWriter(w),
	}
}

// Write делает записи в заголовки
func (c *compressWriter) Write(p []byte) (int, error) {
	n, err := c.zw.Write(p)
	if err != nil {
		return 0, err
	}
	c.Header().Set("Content-Length", strconv.Itoa(n))

	return n, err
}

// WriteHeader делает записи в заголовки ответа
func (c *compressWriter) WriteHeader(statusCode int) {
	c.Header().Set("Content-Encoding", "gzip")
	c.ResponseWriter.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и досылает все данные из буфера.
func (c *compressWriter) Close() error {
	return c.zw.Close()
}

// compressReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные.
type compressReader struct {
	io.ReadCloser
	zr *gzip.Reader
}

// newCompressReader функция конструктор на чтение
func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		ReadCloser: r,
		zr:         zr,
	}, nil
}

// Read позволяет читать данные
func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

// Close закрывает
func (c *compressReader) Close() error {
	return c.zr.Close()
}

// Compress метод по компресии данных
func Compress() gin.HandlerFunc {
	return func(c *gin.Context) {
		ow := c.Writer

		acceptEncoding := c.Request.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			cw := newCompressWriter(c.Writer)
			ow = cw
			defer cw.Close()
		}

		contentEncoding := c.Request.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := newCompressReader(c.Request.Body)
			if err != nil {
				log.Printf("Error compressing: %v", err)
				c.Writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			c.Request.Body = cr
			defer cr.Close()
		}
		c.Writer = ow
		c.Next()

	}
}
