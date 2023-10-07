package fs

import (
	"encoding/json"
	"errors"
	"github.com/EvgeniyBudaev/shortener/internal/models"
	"github.com/EvgeniyBudaev/shortener/internal/store/memory"
	"io"
	"os"
	"strconv"
)

type FSStorage struct {
	path string
	*memory.MemoryStorage
	sr *StorageReader
	sw *StorageWriter
}

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

func (s *FSStorage) PutBatch(urls []models.URLBatchReq) ([]models.URLBatchRes, error) {
	result := make([]models.URLBatchRes, 0)

	for _, url := range urls {
		id, err := s.Put(url.CorrelationID, url.OriginalURL)
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

func (s *FSStorage) Ping() error {
	return nil
}

func (s *FSStorage) DeleteStorageFile() error {
	return os.Remove(s.path)
}

type StorageReader struct {
	file    *os.File
	decoder *json.Decoder
}

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

func (sr *StorageReader) ReadFromFile() (map[string]string, error) {
	records := make(map[string]string)
	for {
		r, err := sr.ReadLine()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}
		records[r.ShortURL] = r.OriginalURL
	}

	return records, nil
}

func (sr *StorageReader) ReadLine() (*models.URLRecord, error) {
	r := models.URLRecord{}
	if err := sr.decoder.Decode(&r); err != nil {
		return nil, err
	}

	return &r, nil
}

type StorageWriter struct {
	file    *os.File
	encoder *json.Encoder
}

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

func (sw *StorageWriter) AppendToFile(r *models.URLRecord) error {
	return sw.encoder.Encode(&r)
}

func (s *FSStorage) Put(id string, url string) (string, error) {
	id, err := s.MemoryStorage.Put(id, url)
	if (err) != nil {
		return "", err
	}
	return id, s.sw.AppendToFile(&models.URLRecord{UUID: strconv.Itoa(s.UrlsCount), OriginalURL: url, ShortURL: id})
}
