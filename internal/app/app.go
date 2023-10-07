package app

import (
	"database/sql"
	"encoding/json"
	"github.com/EvgeniyBudaev/shortener/internal/config"
	"github.com/EvgeniyBudaev/shortener/internal/store"
	"github.com/EvgeniyBudaev/shortener/internal/utils"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"io"
	"log"
	"net/http"
	"net/url"
)

const driverName = "pgx"

type (
	App struct {
		config *config.ServerConfig
		store  *store.Storage
	}

	ShortenReq struct {
		URL string `json:"url"`
	}

	ShortenRes struct {
		Result string `json:"result"`
	}
)

func NewApp(config *config.ServerConfig, storage *store.Storage) *App {
	return &App{
		config: config,
		store:  storage,
	}
}

func (a *App) RedirectURL(c *gin.Context) {
	res := c.Writer
	id := c.Param("id")

	originalURL := a.store.Get(id)

	if originalURL == "" {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	res.Header().Set("Location", originalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

func (a *App) ShortURL(c *gin.Context) {
	req := c.Request
	res := c.Writer

	var originalURL string

	switch req.RequestURI {
	case "/api/shorten":
		var shorten ShortenReq
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

	id, err := utils.GenerateRandomString(8)
	if err != nil {
		log.Printf("Random string generator error: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	resultURL, err := url.JoinPath(a.config.RedirectBaseURL, id)
	if err != nil {
		log.Printf("URL cannot be joined: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	a.store.Put(id, originalURL)

	switch req.RequestURI {
	case "/api/shorten":
		respURL := ShortenRes{
			Result: resultURL,
		}
		resp, err := json.Marshal(respURL)
		if err != nil {
			log.Printf("URL cannot be encoded: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusCreated)
		res.Write(resp)

	case "/":
		res.Header().Set("Content-Type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		if _, err := res.Write([]byte(resultURL)); err != nil {
			log.Printf("Error writing body: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func (a *App) DBPingCheck(c *gin.Context) {
	db, err := sql.Open(driverName, a.config.DatabaseDSN)
	if err != nil {
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer db.Close()
	c.Writer.WriteHeader(http.StatusOK)
}
