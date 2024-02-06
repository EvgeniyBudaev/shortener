// Модуль по работе с файловым хранилищем
package fs

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/EvgeniyBudaev/shortener/internal/models"
	"github.com/EvgeniyBudaev/shortener/internal/store/memory"
	"github.com/gin-gonic/gin"
	"io"
	"os"
	"strconv"
	"sync"
)

// FSStorage описывает структуру файлового хранилища
type FSStorage struct {
	countMutex sync.Mutex
	path       string
	*memory.MemoryStorage
	sr *StorageReader
	sw *StorageWriter
}

// NewFileStorage функция-констукртор
func NewFileStorage(filename string) (*FSStorage, error) {
	sr, err := NewStorageReader(filename)
	if err != nil {
		return nil, err
	}

	records, err := sr.ReadFromFile()
	if err != nil {
		return nil, err
	}

	storage, err := memory.NewMemoryStorage(records)
	if err != nil {
		return nil, err
	}

	sw, err := NewStorageWriter(filename)
	if err != nil {
		return nil, err
	}

	return &FSStorage{
		path:          filename,
		MemoryStorage: storage,
		sr:            sr,
		sw:            sw,
	}, nil
}

// PutBatch метод обновления батча
func (s *FSStorage) PutBatch(ctx *gin.Context, urls []models.URLBatchReq, userID string) ([]models.URLBatchRes, error) {
	result := make([]models.URLBatchRes, 0)

	for _, url := range urls {
		id, err := s.Put(ctx, url.CorrelationID, url.OriginalURL, userID)
		if err != nil {
			return nil, err
		}
		result = append(result, models.URLBatchRes{
			CorrelationID: url.CorrelationID,
			ShortURL:      id,
		})
	}

	return result, nil
}

// Ping метод проверки соединения с БД
func (s *FSStorage) Ping() error {
	return nil
}

// Close метод закрытия соединения
func (s *FSStorage) Close() {
	s.sw.file.Close()
}

// DeleteStorageFile метод удаления файла в файловом хранилище
func (s *FSStorage) DeleteStorageFile() error {
	return os.Remove(s.path)
}

// StorageReader структура хранилища на чтение
type StorageReader struct {
	file    *os.File
	decoder *json.Decoder
}

// NewStorageReader функция-конструктор
func NewStorageReader(filename string) (*StorageReader, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	return &StorageReader{
		file:    file,
		decoder: json.NewDecoder(file),
	}, nil
}

// ReadFromFile метод чтения данных из файла
func (sr *StorageReader) ReadFromFile() (map[string]models.URLRecordMemory, error) {
	records := make(map[string]models.URLRecordMemory)
	for {
		r, err := sr.ReadLine()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		records[r.ShortURL] = models.URLRecordMemory{OriginalURL: r.OriginalURL, UserID: r.UserID}
	}

	return records, nil
}

// ReadLine метод чтения строки в файле
func (sr *StorageReader) ReadLine() (*models.URLRecordFS, error) {
	r := models.URLRecordFS{}
	if err := sr.decoder.Decode(&r); err != nil {
		return nil, err
	}

	return &r, nil
}

// StorageWriter структура хранилища на запись
type StorageWriter struct {
	file    *os.File
	encoder *json.Encoder
}

// NewStorageWriter функция-конструктор
func NewStorageWriter(filename string) (*StorageWriter, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &StorageWriter{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

// AppendToFile метод добавления
func (sw *StorageWriter) AppendToFile(r *models.URLRecordFS) error {
	return sw.encoder.Encode(&r)
}

// Put метод обновления
func (s *FSStorage) Put(ctx *gin.Context, id string, url string, userID string) (string, error) {
	id, err := s.MemoryStorage.Put(ctx, id, url, userID)
	if err != nil {
		return "", err
	}
	s.countMutex.Lock()
	currentCount := s.UrlsCount
	s.countMutex.Unlock()
	return id, s.sw.AppendToFile(
		&models.URLRecordFS{UUID: strconv.Itoa(currentCount), UserID: userID, URLRecord: models.URLRecord{
			OriginalURL: url, ShortURL: id,
		}})
}

// GetStats метод получения
func (s *FSStorage) GetStats() (*models.Stats, error) {
	return nil, fmt.Errorf("not implemented")
}
