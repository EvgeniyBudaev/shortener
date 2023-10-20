package main

import (
	"github.com/EvgeniyBudaev/shortener/internal/compress"
	"github.com/EvgeniyBudaev/shortener/internal/store"
	"log"

	"github.com/EvgeniyBudaev/shortener/internal/app"
	"github.com/EvgeniyBudaev/shortener/internal/config"
	ginLogger "github.com/EvgeniyBudaev/shortener/internal/logger"
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
	r.GET("/ping", a.Ping)

	api := r.Group("/api")
	{
		api.POST("/shorten", a.ShortURL)
		api.POST("/shorten/batch", a.ShortenBatch)
	}

	return r
}

func main() {
	initConfig, err := config.InitFlags()
	if err != nil {
		log.Fatal(err)
	}

	storage, err := store.NewStore(initConfig)
	if err != nil {
		log.Fatal(err)
	}

	appInit := app.NewApp(initConfig, storage)

	r := setupRouter(appInit)
	log.Fatal(r.Run(initConfig.FlagRunAddr))
}
