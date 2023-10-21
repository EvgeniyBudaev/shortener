package memory

import (
	"github.com/EvgeniyBudaev/shortener/internal/models"
	"github.com/gin-gonic/gin"
	"sync"
)

type MemoryStorage struct {
	mux       *sync.Mutex
	urls      map[string]models.URLRecordMemory
	UrlsCount int
}

func NewMemoryStorage(records map[string]models.URLRecordMemory) (*MemoryStorage, error) {
	return &MemoryStorage{
		mux:       &sync.Mutex{},
		urls:      records,
		UrlsCount: len(records),
	}, nil
}

func (s *MemoryStorage) Put(ctx *gin.Context, id string, url string, userID string) (string, error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.urls[id] = models.URLRecordMemory{
		OriginalURL: url,
		UserID:      userID,
	}
	s.UrlsCount += 1
	return id, nil
}

func (s *MemoryStorage) Get(ctx *gin.Context, id string) (string, error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	originalURL := s.urls[id]
	return originalURL.OriginalURL, nil
}

func (s *MemoryStorage) GetAllByUserID(ctx *gin.Context, userID string) ([]models.URLRecord, error) {
	result := make([]models.URLRecord, 0)
	for id, url := range s.urls {
		if url.UserID == userID {
			s.mux.Lock()
			defer s.mux.Unlock()

			result = append(result, models.URLRecord{
				ShortURL:    id,
				OriginalURL: url.OriginalURL,
			})
		}
	}
	return result, nil
}

func (s *MemoryStorage) DeleteMany(ctx *gin.Context, ids models.DeleteUserURLsReq, userID string) error {
	for _, id := range ids {
		if url, ok := s.urls[id]; ok && url.UserID == userID {
			delete(s.urls, id)
		}
	}
	return nil
}

func (s *MemoryStorage) PutBatch(ctx *gin.Context, urls []models.URLBatchReq, userID string) ([]models.URLBatchRes, error) {
	result := make([]models.URLBatchRes, 0)

	for _, url := range urls {
		id, err := s.Put(ctx, url.CorrelationID, url.OriginalURL, userID)
		if err != nil {
			return nil, err
		}
		result = append(result, models.URLBatchRes{
			CorrelationID: id,
			ShortURL:      id,
		})
	}

	return result, nil
}

func (s *MemoryStorage) Ping() error {
	return nil
}

func (s *MemoryStorage) Close() {
}
