package main

import (
	"log"

	"github.com/EvgeniyBudaev/shortener/internal/app"
	"github.com/EvgeniyBudaev/shortener/internal/config"
	ginLogger "github.com/EvgeniyBudaev/shortener/internal/logger"
	"github.com/EvgeniyBudaev/shortener/internal/store"
	"github.com/gin-gonic/gin"
)

func setupRouter(a *app.App) *gin.Engine {
	r := gin.New()
	r.Use(ginLogger.Logger())

	r.GET("/:id", a.RedirectURL)
	r.POST("/", a.ShortURL)

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
	intApp := app.NewApp(initConfig, store.NewStorage())

	r := setupRouter(intApp)
	log.Fatal(r.Run(initConfig.FlagRunAddr))
}
