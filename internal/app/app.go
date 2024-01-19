// Модуль приложения
package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/EvgeniyBudaev/shortener/internal/config"
	"github.com/EvgeniyBudaev/shortener/internal/middleware/auth"
	"github.com/EvgeniyBudaev/shortener/internal/models"
	"github.com/EvgeniyBudaev/shortener/internal/store/postgres"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"io"
	"log"
	"net/http"
	"net/url"
)

const (
	rootPath = "/"
	pingPath = "/ping"
)

// Store Интерфейс содержит все необходимые методы для работы сервиса.
type Store interface {
	Get(ctx *gin.Context, id string) (string, error)
	GetAllByUserID(ctx *gin.Context, userID string) ([]models.URLRecord, error)
	DeleteMany(ctx *gin.Context, ids models.DeleteUserURLsReq, userID string) error
	Put(ctx *gin.Context, id string, shortURL string, userID string) (string, error)
	PutBatch(ctx *gin.Context, data []models.URLBatchReq, userID string) ([]models.URLBatchRes, error)
	Ping() error
}

// App структура приложения
type App struct {
	Config *config.ServerConfig
	store  Store
	logger *zap.SugaredLogger
}

// NewApp конструктор приложения
func NewApp(config *config.ServerConfig, store Store, logger *zap.SugaredLogger) *App {
	return &App{
		Config: config,
		store:  store,
		logger: logger,
	}
}

// DeleteUserRecords удаление записей по пользователю
func (a *App) DeleteUserRecords(c *gin.Context) {
	req := c.Request
	res := c.Writer
	userID := c.GetString(auth.UserIDKey)
	batch := make(models.DeleteUserURLsReq, 0)

	deleteChan := make(chan models.DeleteUserURLsReq)
	done := make(chan bool)

	deleteWorker := func() {
		for batch := range deleteChan {
			err := a.store.DeleteMany(c, batch, userID)
			if err != nil {
				log.Printf("error deleting: %v", err)
			}
		}
		done <- true
	}

	go deleteWorker()

	batchCh := make(chan models.DeleteUserURLsReq)
	go func() {
		if err := json.NewDecoder(req.Body).Decode(&batch); err != nil {
			log.Printf("Body cannot be decoded: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		batchCh <- batch
		close(batchCh)
	}()

	go func() {
		for batch := range batchCh {
			deleteChan <- batch
		}
		close(deleteChan)
	}()

	res.WriteHeader(http.StatusAccepted)
}

// GetUserRecords получение всех записей пользователя
func (a *App) GetUserRecords(c *gin.Context) {
	res := c.Writer
	userID := c.GetString(auth.UserIDKey)

	records, err := a.store.GetAllByUserID(c, userID)
	if err != nil {
		log.Printf("Error getting all user urls: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(records) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	for idx, urlObj := range records {
		resultURL, err := url.JoinPath(a.Config.RedirectBaseURL, urlObj.ShortURL)
		if err != nil {
			log.Printf("URL cannot be joined: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		records[idx].ShortURL = resultURL
	}

	res.Header().Add("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(res).Encode(records); err != nil {
		log.Printf("Error writing response in JSON: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// RedirectURL перенаправление на URL
func (a *App) RedirectURL(c *gin.Context) {
	res := c.Writer
	id := c.Param("id")

	originalURL, err := a.store.Get(c, id)
	if err != nil {
		if errors.Is(err, postgres.ErrURLDeleted) {
			res.WriteHeader(http.StatusGone)
			return
		} else {
			log.Printf("Error getting original URL: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if originalURL == "" {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	res.Header().Set("Location", originalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

// ShortenBatch метод по работе с батчем
func (a *App) ShortenBatch(c *gin.Context) {
	req := c.Request
	res := c.Writer
	userID := c.GetString(auth.UserIDKey)

	batch := make([]models.URLBatchReq, 0)
	if err := json.NewDecoder(req.Body).Decode(&batch); err != nil {
		log.Printf("Body cannot be decoded: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	result, err := a.store.PutBatch(c, batch, userID)
	if err != nil {
		log.Printf("Cant put batch: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	for idx, urlObj := range result {
		resultURL, err := url.JoinPath(a.Config.RedirectBaseURL, urlObj.CorrelationID)
		if err != nil {
			log.Printf("URL cannot be joined: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		result[idx].ShortURL = resultURL
	}

	res.WriteHeader(http.StatusCreated)
	res.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(res).Encode(result); err != nil {
		log.Printf("Error writing response in JSON: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// ShortURL метод по сокращению URL
func (a *App) ShortURL(c *gin.Context) {
	req := c.Request
	res := c.Writer
	userID := c.GetString(auth.UserIDKey)

	var originalURL string

	switch req.RequestURI {
	case "/api/shorten":
		var shorten models.ShortenReq
		if err := json.NewDecoder(req.Body).Decode(&shorten); err != nil {
			log.Printf("Body cannot be decoded: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		originalURL = shorten.URL
	case "/":
		body, err := io.ReadAll(req.Body)
		if err != nil {
			log.Printf("Body cannot be read: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		originalURL = string(body)
	}

	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		log.Printf("Random string generator error: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	id := hex.EncodeToString(b)

	id, err = a.store.Put(c, id, originalURL, userID)
	if err != nil {
		if errors.Is(err, postgres.ErrDBInsertConflict) {
			res.WriteHeader(http.StatusConflict)
		} else {
			log.Printf("Error saving data: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		res.WriteHeader(http.StatusCreated)
	}

	resultURL, err := url.JoinPath(a.Config.RedirectBaseURL, id)
	if err != nil {
		log.Printf("URL cannot be joined: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch req.RequestURI {
	case "/api/shorten":
		respURL := models.ShortenRes{
			Result: resultURL,
		}
		resp, err := json.Marshal(respURL)
		if err != nil {
			log.Printf("URL cannot be encoded: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json")
		res.Write(resp)

	case "/":
		res.Header().Set("Content-Type", "text/plain")
		if _, err := res.Write([]byte(resultURL)); err != nil {
			log.Printf("Error writing body: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

// Ping метод по проверке соединения с БД
func (a *App) Ping(c *gin.Context) {
	if err := a.store.Ping(); err != nil {
		log.Printf("Error opening connection to DB: %v", err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Writer.WriteHeader(http.StatusOK)
}
