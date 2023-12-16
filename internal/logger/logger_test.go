package logger

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"net/http/httptest"
)

func ExampleLogger() {
	// Создаем экземпляр маршрутизатора Gin
	r := gin.New()

	// Создаем экземпляр middleware Logger
	ginLoggerMiddleware, err := Logger()
	if err != nil {
		log.Fatal(err)
	}

	// Добавляем Logger как middleware
	r.Use(ginLoggerMiddleware)

	// Добавляем обработчик для тестирования
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// Создаем тестовый HTTP-запрос
	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()

	// Выполняем запрос с помощью созданного маршрутизатора
	r.ServeHTTP(w, req)

	// Примеры вывода логов будут доступны в документации
}
