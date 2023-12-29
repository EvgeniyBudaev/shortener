// Модуль логирования запросов.
package logger

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger Получение middleware функции, которая будет логгировать входящие запросы.
func Logger() (gin.HandlerFunc, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	return func(c *gin.Context) {
		uri := c.Request.RequestURI
		method := c.Request.Method
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		t := time.Now()
		c.Next()
		duration := time.Since(t)

		sugar.Infoln(
			"URI", uri,
			"Method", method,
			"Duration", duration,
			"Status", c.Writer.Status(),
			"Size", c.Writer.Size(),
		)
		sugar.Debugln("Data", string(body))
	}, nil
}

func NewLogger() (*zap.SugaredLogger, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("error creating logger: %w", err)
	}

	return logger.Sugar(), nil
}
