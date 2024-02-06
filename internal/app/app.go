// Модуль приложения
package app

import (
	"encoding/json"
	"errors"
	"github.com/EvgeniyBudaev/shortener/internal/auth"
	"github.com/EvgeniyBudaev/shortener/internal/config"
	"github.com/EvgeniyBudaev/shortener/internal/logic"
	"github.com/EvgeniyBudaev/shortener/internal/models"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"io"
	"net/http"
)

const (
	slugLength      = 4
	applicationJSON = "application/json"
	textPlain       = "text/plain"
	contentType     = "Content-Type"
	location        = "Location"

	rootPath       = "/"
	pingPath       = "/ping"
	apiShortenPath = "/api/shorten"

	ErrorDecodeBody  = "Body cannot be decoded: %v"
	ErrorWritingBody = "Error writing body: %v"
)

// Store Интерфейс содержит все необходимые методы для работы сервиса.
type Store interface {
	Get(id string) (string, error)
	GetStats() (*models.Stats, error)
	GetAllByUserID(ctx *gin.Context, userID string) ([]models.URLRecord, error)
	DeleteMany(ctx *gin.Context, ids models.DeleteUserURLsReq, userID string) error
	Put(ctx *gin.Context, id string, shortURL string, userID string) (string, error)
	PutBatch(ctx *gin.Context, data []models.URLBatchReq, userID string) ([]models.URLBatchRes, error)
	Ping() error
}

// App структура приложения
type App struct {
	Config    *config.ServerConfig
	Logger    *zap.SugaredLogger
	coreLogic *logic.CoreLogic
}

// NewApp конструктор приложения
func NewApp(config *config.ServerConfig, coreLogic *logic.CoreLogic, logger *zap.SugaredLogger) *App {
	return &App{
		Config:    config,
		coreLogic: coreLogic,
		Logger:    logger,
	}
}

// DeleteUserRecords удаление записей по пользователю
func (a *App) DeleteUserRecords(c *gin.Context) {
	req := c.Request
	res := c.Writer
	userID := c.GetString(auth.UserIDKey)
	batch := make(models.DeleteUserURLsReq, 0)
	if err := json.NewDecoder(req.Body).Decode(&batch); err != nil {
		a.Logger.Errorf(ErrorDecodeBody, err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	go func() {
		if err := a.coreLogic.DeleteUserRecords(c, userID, batch); err != nil {
			a.Logger.Errorf("error deleting: %v", err)
		}
	}()
	res.WriteHeader(http.StatusAccepted)
}

// GetUserRecords получение всех записей пользователя
func (a *App) GetUserRecords(c *gin.Context) {
	res := c.Writer
	userID := c.GetString(auth.UserIDKey)
	records, err := a.coreLogic.GetUserRecords(c, userID)
	if err != nil {
		if errors.Is(err, logic.ErrNoContent) {
			res.WriteHeader(http.StatusNoContent)
			return
		}
		a.Logger.Errorf("Error getting all user urls: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, records)
}

// RedirectURL перенаправление на URL
func (a *App) RedirectURL(c *gin.Context) {
	res := c.Writer
	id := c.Param("id")
	originalURL, err := a.coreLogic.GetOriginalURL(c, id)
	if err != nil {
		if errors.Is(err, logic.ErrIsDeleted) {
			res.WriteHeader(http.StatusGone)
			return
		}
		if errors.Is(err, logic.ErrNotFound) {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		a.Logger.Errorf("Error getting original URL: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, originalURL)
}

// ShortenBatch метод по работе с батчем
func (a *App) ShortenBatch(c *gin.Context) {
	req := c.Request
	res := c.Writer
	userID := c.GetString(auth.UserIDKey)
	batch := make([]models.URLBatchReq, 0)
	if err := json.NewDecoder(req.Body).Decode(&batch); err != nil {
		a.Logger.Errorf(ErrorDecodeBody, err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	result, err := a.coreLogic.ShortenBatch(c, userID, batch)
	if err != nil {
		a.Logger.Errorf("Cant put batch: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusCreated, result)
}

// ShortURL метод по сокращению URL
func (a *App) ShortURL(c *gin.Context) {
	req := c.Request
	res := c.Writer
	userID := c.GetString(auth.UserIDKey)

	var originalURL string

	switch req.RequestURI {
	case apiShortenPath:
		var shorten models.ShortenReq
		if err := json.NewDecoder(req.Body).Decode(&shorten); err != nil {
			a.Logger.Errorf(ErrorDecodeBody, err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		originalURL = shorten.URL
	case rootPath:
		body, err := io.ReadAll(req.Body)
		if err != nil {
			a.Logger.Errorf("Body cannot be read: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		originalURL = string(body)
	}
	resultURL, err := a.coreLogic.ShortenURL(c, userID, originalURL)
	if err != nil {
		if errors.Is(err, logic.ErrConflict) {
			res.WriteHeader(http.StatusConflict)
			return
		}
		a.Logger.Errorf("Error saving data: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusCreated)
	switch req.RequestURI {
	case apiShortenPath:
		respURL := models.ShortenRes{
			Result: resultURL,
		}
		resp, err := json.Marshal(respURL)
		if err != nil {
			a.Logger.Errorf("URL cannot be encoded: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		res.Header().Set(contentType, applicationJSON)
		if _, err := res.Write(resp); err != nil {
			a.Logger.Errorf(ErrorWritingBody, err)
		}
	case rootPath:
		res.Header().Set(contentType, textPlain)
		if _, err := res.Write([]byte(resultURL)); err != nil {
			a.Logger.Errorf("Error writing body: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

// Ping метод по проверке соединения с БД
func (a *App) Ping(c *gin.Context) {
	if err := a.coreLogic.Ping(c); err != nil {
		a.Logger.Errorf("Error opening connection to DB: %v", err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Writer.WriteHeader(http.StatusOK)
}

// GetStats метод получения
func (a *App) GetStats(c *gin.Context) {
	stats, err := a.coreLogic.GetStats(c)
	if err != nil {
		a.Logger.Errorf("error getting service stats: %v", err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, stats)
}
