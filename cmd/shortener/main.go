package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/EvgeniyBudaev/shortener/internal/store"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/EvgeniyBudaev/shortener/internal/app"
	"github.com/EvgeniyBudaev/shortener/internal/config"
	ginLogger "github.com/EvgeniyBudaev/shortener/internal/logger"
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

	appInit := app.NewApp(initConfig, storage, logger)

	r, err := appInit.SetupRouter()
	if err != nil {
		logger.Fatal(err)
	}
	srv := http.Server{
		Addr:    initConfig.FlagRunAddr,
		Handler: r,
	}

	go func(errs chan<- error) {
		if initConfig.EnableHTTPS {
			_, errCert := os.ReadFile(initConfig.TLSCertPath)
			_, errKey := os.ReadFile(initConfig.TLSKeyPath)

			if errors.Is(errCert, os.ErrNotExist) || errors.Is(errKey, os.ErrNotExist) {
				privateKey, certBytes, err := app.CreateCertificates(logger.Named("certs-builder"))
				if err != nil {
					errs <- fmt.Errorf("error creating tls certs: %w", err)
					return
				}

				if err := app.WriteCertificates(certBytes, initConfig.TLSCertPath, privateKey, initConfig.TLSKeyPath, logger); err != nil {
					errs <- fmt.Errorf("error writing tls certs: %w", err)
					return
				}
			}

			if err := srv.ListenAndServeTLS(initConfig.TLSCertPath, initConfig.TLSKeyPath); err != nil {
				if errors.Is(err, http.ErrServerClosed) {
					return
				}
				errs <- fmt.Errorf("run tls server has failed: %w", err)
				return
			}
		}

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
		logger.Error(err)
		cancelCtx()
	}
}
