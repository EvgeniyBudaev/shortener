package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/EvgeniyBudaev/shortener/internal/auth"
	"github.com/EvgeniyBudaev/shortener/internal/compress"
	"github.com/EvgeniyBudaev/shortener/internal/store"
	"github.com/gin-contrib/pprof"
	"log"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/EvgeniyBudaev/shortener/internal/app"
	"github.com/EvgeniyBudaev/shortener/internal/config"
	ginLogger "github.com/EvgeniyBudaev/shortener/internal/logger"
	"github.com/gin-gonic/gin"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

const (
	timeoutServerShutdown = time.Second * 5
	timeoutShutdown       = time.Second * 10
)

func setupRouter(a *app.App) *gin.Engine {
	r := gin.New()
	pprof.Register(r)
	ginLoggerMiddleware, err := ginLogger.Logger()
	if err != nil {
		log.Fatal(err)
	}
	r.Use(ginLoggerMiddleware)
	r.Use(auth.AuthMiddleware(a.Config.Seed))
	r.Use(compress.Compress())

	r.GET("/:id", a.RedirectURL)
	r.POST("/", a.ShortURL)
	r.GET("/ping", a.Ping)

	api := r.Group("/api")
	{
		api.POST("/shorten", a.ShortURL)
		api.POST("/shorten/batch", a.ShortenBatch)

		api.GET("/user/urls", a.GetUserRecords)
		api.DELETE("/user/urls", a.DeleteUserRecords)
	}

	return r
}

func main() {
	ctx, cancelCtx := signal.NotifyContext(context.Background(), syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	logger, err := ginLogger.NewLogger()
	if err != nil {
		log.Fatal(err)
	}

	logger.Infof("Build version: %s", buildVersion)
	logger.Infof("Build date: %s", buildDate)
	logger.Infof("Build commit: %s", buildCommit)

	defer cancelCtx()

	appConfig, err := config.ParseFlags()
	if err != nil {
		log.Fatal(err)
	}

	storage, err := store.NewStore(ctx, appConfig)
	if err != nil {
		log.Fatal(err)
	}

	wg := &sync.WaitGroup{}
	defer func() {
		wg.Wait()
	}()

	componentsErrs := make(chan error, 1)

	appInit := app.NewApp(appConfig, storage)

	r := setupRouter(appInit)
	srv := http.Server{
		Addr:    appConfig.FlagRunAddr,
		Handler: r,
	}

	go func(errs chan<- error) {
		if appConfig.EnableHTTPS {
			certFilePath := "./certs/cert.pem"
			rsaFilePath := "./certs/private.pem"
			certsExist, err := app.CheckIfCertificatesExist(certFilePath, rsaFilePath)
			if err != nil {
				log.Fatal(err)
			}
			if !certsExist {
				// Если файлы не существуют, создаем новые сертификаты
				if err := app.CreateCertificates(); err != nil {
					errs <- fmt.Errorf("error creating tls certs: %w", err)
				}
			}
			if err := srv.ListenAndServeTLS(certFilePath, rsaFilePath); err != nil {
				if errors.Is(err, http.ErrServerClosed) {
					return
				}
				errs <- fmt.Errorf("run tls server has failed: %w", err)
			}
		} else {
			if err := srv.ListenAndServe(); err != nil {
				if errors.Is(err, http.ErrServerClosed) {
					return
				}
				errs <- fmt.Errorf("run server has failed: %w", err)
			}
		}
	}(componentsErrs)

	wg.Add(1)
	go func() {
		defer log.Print("server has been shutdown and close DB")
		defer wg.Done()
		<-ctx.Done()

		shutdownTimeoutCtx, cancelShutdownTimeoutCtx := context.WithTimeout(context.Background(), timeoutServerShutdown)
		defer cancelShutdownTimeoutCtx()
		if err := srv.Shutdown(shutdownTimeoutCtx); err != nil {
			log.Printf("an error occurred during server shutdown: %v", err)
		}
		storage.Close()
	}()

	select {
	case <-ctx.Done():
	case err := <-componentsErrs:
		log.Print(err)
		cancelCtx()
	}
}
