package main

import (
	"github.com/EvgeniyBudaev/shortener/internal/compress"
	"log"

	"github.com/EvgeniyBudaev/shortener/internal/app"
	"github.com/EvgeniyBudaev/shortener/internal/config"
	ginLogger "github.com/EvgeniyBudaev/shortener/internal/logger"
	"github.com/EvgeniyBudaev/shortener/internal/store"
	"github.com/gin-gonic/gin"
)

func setupRouter(a *app.App) *gin.Engine {
	r := gin.New()
	ginLoggerMiddleware, err := ginLogger.Logger()
	if err != nil {
		log.Fatal(err)
	}
	r.Use(ginLoggerMiddleware)
	r.Use(compress.Compress())

	r.GET("/:id", a.RedirectURL)
	r.POST("/", a.ShortURL)
	r.GET("/ping", a.DBPingCheck)

	api := r.Group("/api")
	{
		api.POST("/shorten", a.ShortURL)
	}

	return r
}

func main() {
	initConfig, err := config.InitFlags()
	if err != nil {
		log.Fatal(err)
	}
	storage, err := store.NewStorage(initConfig.FileStoragePath)
	if err != nil {
		log.Fatal(err)
	}
	intApp := app.NewApp(initConfig, storage)

	r := setupRouter(intApp)
	log.Fatal(r.Run(initConfig.FlagRunAddr))
}
