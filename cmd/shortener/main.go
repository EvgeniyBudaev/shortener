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
	var storage app.Store
	if initConfig.DatabaseDSN != "" {
		s, err := store.NewDBStore(initConfig.DatabaseDSN)
		if err != nil {
			log.Fatal(err)
		}
		storage.Get = s.Get
		storage.Put = s.Put
		defer s.Close()
	} else {
		s, err := store.NewStorage(initConfig.FileStoragePath)
		if err != nil {
			log.Fatal(err)
		}
		storage.Get = s.Get
		storage.Put = s.Put
	}
	intApp := app.NewApp(initConfig, &storage)

	r := setupRouter(intApp)
	log.Fatal(r.Run(initConfig.FlagRunAddr))
}
