// Модуль работает как единая точка входа для создания хранилища сервиса.
package store

import (
	"context"
	"github.com/EvgeniyBudaev/shortener/internal/config"
	"github.com/EvgeniyBudaev/shortener/internal/models"
	"github.com/EvgeniyBudaev/shortener/internal/store/fs"
	"github.com/EvgeniyBudaev/shortener/internal/store/memory"
	"github.com/EvgeniyBudaev/shortener/internal/store/postgres"
	"github.com/gin-gonic/gin"
)

// Store Интерфейс содержит все необходимые методы для работы сервиса.
type Store interface {
	Get(ctx *gin.Context, id string) (string, error)
	GetAllByUserID(ctx *gin.Context, userID string) ([]models.URLRecord, error)
	DeleteMany(ctx *gin.Context, ids models.DeleteUserURLsReq, userID string) error
	Put(ctx *gin.Context, id string, shortURL string, userID string) (string, error)
	PutBatch(ctx *gin.Context, data []models.URLBatchReq, userID string) ([]models.URLBatchRes, error)
	Ping() error
	Close()
}

// NewStore Функция получения конкретной реализации интерфейса.
// Приоритет выбора: база данных, сохранение в файл, внутрення память.
func NewStore(ctx context.Context, conf *config.ServerConfig) (Store, error) {
	if conf.DatabaseDSN != "" {
		return postgres.NewPostgresStore(ctx, conf.DatabaseDSN)
	}
	if conf.FileStoragePath != "" {
		return fs.NewFileStorage(conf.FileStoragePath)
	}
	return memory.NewMemoryStorage(make(map[string]models.URLRecordMemory))
}
