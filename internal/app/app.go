package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/EvgeniyBudaev/shortener/internal/auth"
	"github.com/EvgeniyBudaev/shortener/internal/config"
	"github.com/EvgeniyBudaev/shortener/internal/models"
	"github.com/EvgeniyBudaev/shortener/internal/store/postgres"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
)

type Store interface {
	Get(ctx *gin.Context, id string) (string, error)
	GetAllByUserID(ctx *gin.Context, userID string) ([]models.URLRecord, error)
	DeleteMany(ctx *gin.Context, ids models.DeleteUserURLsReq, userID string) error
	Put(ctx *gin.Context, id string, shortURL string, userID string) (string, error)
	PutBatch(ctx *gin.Context, data []models.URLBatchReq, userID string) ([]models.URLBatchRes, error)
	Ping() error
}

type App struct {
	Config *config.ServerConfig
	store  Store
}

func NewApp(config *config.ServerConfig, store Store) *App {
	return &App{
		Config: config,
		store:  store,
	}
}

func (a *App) DeleteUserRecords(c *gin.Context) {
	req := c.Request
	res := c.Writer
	userID := c.GetString(auth.UserIDKey)

	batch := make(models.DeleteUserURLsReq, 0)
	if err := json.NewDecoder(req.Body).Decode(&batch); err != nil {
		log.Printf("Body cannot be decoded: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	go a.ExecDeleteUserRecords(c, batch, userID)

	res.WriteHeader(http.StatusAccepted)
}

func (a *App) ExecDeleteUserRecords(c *gin.Context, batch models.DeleteUserURLsReq, userID string) {
	var countJob = 5
	jobCh := make(chan string, 1)
	var wg sync.WaitGroup
	for i := 0; i < countJob; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _ = range jobCh {
				err := a.store.DeleteMany(c, batch, userID)
				if err != nil {
					log.Printf("error deleting: %v", err)
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(jobCh)
	}()
}

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

func (a *App) Ping(c *gin.Context) {
	if err := a.store.Ping(); err != nil {
		log.Printf("Error opening connection to DB: %v", err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Writer.WriteHeader(http.StatusOK)
}
