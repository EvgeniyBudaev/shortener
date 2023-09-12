package main

import (
	"log"

	"github.com/EvgeniyBudaev/shortener/internal/app"
	"github.com/EvgeniyBudaev/shortener/internal/config"
	"github.com/EvgeniyBudaev/shortener/internal/store"
	"github.com/gin-gonic/gin"
)

func setupRouter(a *app.App) *gin.Engine {
	r := gin.Default()

	r.GET("/:id", a.RedirectURL)
	r.POST("/", a.ShortURL)

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
