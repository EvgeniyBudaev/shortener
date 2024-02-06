package logic

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/url"

	"github.com/EvgeniyBudaev/shortener/internal/config"
	"github.com/EvgeniyBudaev/shortener/internal/models"
	"github.com/EvgeniyBudaev/shortener/internal/store/postgres"
	"go.uber.org/zap"
)

const (
	slugLength      = 4
	applicationJSON = "application/json"
	textPlain       = "text/plain"
	contentType     = "Content-Type"
	location        = "Location"

	ErrorJoinURL     = "URL cannot be joined: %v"
	ErrorDecodeBody  = "Body cannot be decoded: %v"
	ErrorEncodeBody  = "Error writing response in JSON: %v"
	ErrorWritingBody = "Error writing body: %v"
)

var (
	ErrNoContent = errors.New("no content")
	ErrNotFound  = errors.New("not found")
	ErrIsDeleted = errors.New("deleted")
	ErrConflict  = errors.New("conflict")
)

type Store interface {
	Get(ctx *gin.Context, id string) (string, error)
	GetStats() (*models.Stats, error)
	GetAllByUserID(ctx *gin.Context, userID string) ([]models.URLRecord, error)
	DeleteMany(ctx *gin.Context, ids models.DeleteUserURLsReq, userID string) error
	Put(ctx *gin.Context, id string, shortURL string, userID string) (string, error)
	PutBatch(ctx *gin.Context, data []models.URLBatchReq, userID string) ([]models.URLBatchRes, error)
	Ping() error
}

type CoreLogic struct {
	config *config.ServerConfig
	store  Store
	logger *zap.SugaredLogger
}

func NewCoreLogic(config *config.ServerConfig, store Store, logger *zap.SugaredLogger) *CoreLogic {
	return &CoreLogic{
		config: config,
		store:  store,
		logger: logger,
	}
}

func (cl *CoreLogic) DeleteUserRecords(ctx *gin.Context, userID string, urls models.DeleteUserURLsReq) error {
	if err := cl.store.DeleteMany(ctx, urls, userID); err != nil {
		err = fmt.Errorf("error deleting: %w", err)
		cl.logger.Error(err)
		return err
	}

	return nil
}

func (cl *CoreLogic) GetUserRecords(ctx *gin.Context, userID string) ([]models.URLRecord, error) {
	records, err := cl.store.GetAllByUserID(ctx, userID)
	if err != nil {
		err = fmt.Errorf("error getting all user urls: %w", err)
		cl.logger.Error(err)
		return nil, err
	}

	if len(records) == 0 {
		return nil, ErrNoContent
	}

	for idx, urlObj := range records {
		resultURL, err := url.JoinPath(cl.config.RedirectBaseURL, urlObj.ShortURL)
		if err != nil {
			err = fmt.Errorf(ErrorJoinURL, err)
			cl.logger.Error(err)
			return nil, err
		}
		records[idx].ShortURL = resultURL
	}

	return records, nil
}

func (cl *CoreLogic) GetOriginalURL(ctx *gin.Context, shortURL string) (string, error) {
	originalURL, err := cl.store.Get(ctx, shortURL)
	if err != nil {
		if errors.Is(err, postgres.ErrURLDeleted) {
			return "", ErrIsDeleted
		}

		err = fmt.Errorf("error getting original URL: %w", err)
		cl.logger.Error(err)
		return "", err
	}

	if originalURL == "" {
		return "", ErrNotFound
	}

	return originalURL, nil
}

func (cl *CoreLogic) ShortenBatch(
	ctx *gin.Context,
	userID string,
	batchURLsReq []models.URLBatchReq,
) ([]models.URLBatchRes, error) {
	result, err := cl.store.PutBatch(ctx, batchURLsReq, userID)
	if err != nil {
		err := fmt.Errorf("cant put batch: %w", err)
		cl.logger.Error(err)
		return nil, err
	}

	for idx, urlObj := range result {
		resultURL, err := url.JoinPath(cl.config.RedirectBaseURL, urlObj.CorrelationID)
		if err != nil {
			err := fmt.Errorf(ErrorJoinURL, err)
			cl.logger.Error(err)
			return nil, err
		}
		result[idx].ShortURL = resultURL
	}

	return result, nil
}

func (cl *CoreLogic) ShortenURL(ctx *gin.Context, userID string, originalURL string) (string, error) {
	b := make([]byte, slugLength)
	_, err := rand.Read(b)
	if err != nil {
		err := fmt.Errorf("random string generator error: %w", err)
		cl.logger.Error(err)
		return "", err
	}
	id := hex.EncodeToString(b)

	id, err = cl.store.Put(ctx, id, originalURL, userID)
	if err != nil {
		if errors.Is(err, postgres.ErrDBInsertConflict) {
			return "", ErrConflict
		} else {
			err := fmt.Errorf("error saving data: %w", err)
			cl.logger.Error(err)
			return "", err
		}
	}

	resultURL, err := url.JoinPath(cl.config.RedirectBaseURL, id)
	if err != nil {
		err := fmt.Errorf(ErrorJoinURL, err)
		cl.logger.Error(err)
		return "", err
	}

	return resultURL, nil
}

func (cl *CoreLogic) Ping(ctx context.Context) error {
	if err := cl.store.Ping(); err != nil {
		err := fmt.Errorf("error opening connection to DB: %w", err)
		cl.logger.Error(err)
		return err
	}

	return nil
}

func (cl *CoreLogic) GetStats(ctx context.Context) (*models.Stats, error) {
	stats, err := cl.store.GetStats()
	if err != nil {
		err := fmt.Errorf("error getting service stats: %w", err)
		cl.logger.Error(err)

		return nil, err
	}

	return stats, nil
}
