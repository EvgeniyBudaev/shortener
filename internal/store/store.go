package store

import (
	"github.com/EvgeniyBudaev/shortener/internal/config"
	"github.com/EvgeniyBudaev/shortener/internal/models"
	"github.com/EvgeniyBudaev/shortener/internal/store/fs"
	"github.com/EvgeniyBudaev/shortener/internal/store/memory"
	"github.com/EvgeniyBudaev/shortener/internal/store/postgres"
	"github.com/gin-gonic/gin"
)

type Store interface {
	Get(ctx *gin.Context, id string) (string, error)
	Put(ctx *gin.Context, id string, shortURL string) (string, error)
	PutBatch(*gin.Context, []models.URLBatchReq) ([]models.URLBatchRes, error)
	Ping() error
}

func NewStore(conf *config.ServerConfig) (Store, error) {
	if conf.DatabaseDSN != "" {
		return postgres.NewPostgresStore(conf.DatabaseDSN)
	}
	if conf.FileStoragePath != "" {
		return fs.NewFileStorage(conf.FileStoragePath)
	}
	return memory.NewMemoryStorage(make(map[string]string))
}
