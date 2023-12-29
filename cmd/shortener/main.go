package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/EvgeniyBudaev/shortener/internal/auth"
	"github.com/EvgeniyBudaev/shortener/internal/compress"
	"github.com/EvgeniyBudaev/shortener/internal/staticlint/osexitchecker"
	"github.com/EvgeniyBudaev/shortener/internal/store"
	"github.com/gin-contrib/pprof"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/staticcheck"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
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
	analyzers := make([]*analysis.Analyzer, 0)

	for _, v := range staticcheck.Analyzers {
		analyzers = append(analyzers, v.Analyzer)
	}

	analyzers = append(analyzers,
		osexitchecker.Analyzer,
		defers.Analyzer,
		nilfunc.Analyzer,
		nilness.Analyzer,
		printf.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		structtag.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unusedresult.Analyzer)

	multichecker.Main(
		analyzers...,
	)

	ctx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt)

	logger, err := ginLogger.NewLogger()
	if err != nil {
		log.Fatal(err)
	}

	logger.Infof("Build version: %s", buildVersion)
	logger.Infof("Build date: %s", buildDate)
	logger.Infof("Build commit: %s", buildCommit)

	defer cancelCtx()

	initConfig, err := config.ParseFlags()
	if err != nil {
		log.Fatal(err)
	}

	storage, err := store.NewStore(ctx, initConfig)
	if err != nil {
		log.Fatal(err)
	}

	wg := &sync.WaitGroup{}
	defer func() {
		wg.Wait()
	}()

	componentsErrs := make(chan error, 1)

	appInit := app.NewApp(initConfig, storage)

	r := setupRouter(appInit)
	srv := http.Server{
		Addr:    initConfig.FlagRunAddr,
		Handler: r,
	}

	go func(errs chan<- error) {
		if err := srv.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}
			errs <- fmt.Errorf("run server has failed: %w", err)
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
