package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/EvgeniyBudaev/shortener/internal/app"
	"github.com/EvgeniyBudaev/shortener/internal/models"
	"github.com/EvgeniyBudaev/shortener/internal/store/fs"
	"github.com/EvgeniyBudaev/shortener/internal/utils"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EvgeniyBudaev/shortener/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirectURL(t *testing.T) {
	type args struct {
		urls           map[string]string
		shortURL       string
		originalURL    string
		shouldRedirect bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "simple redirect",
			args: args{
				urls: map[string]string{
					"1": "http://test.ru",
				},
				originalURL:    "http://test.ru",
				shortURL:       "/1",
				shouldRedirect: true,
			},
		},
		{
			name: "error short url not found",
			args: args{
				urls: map[string]string{
					"1": "http://test.ru",
				},
				originalURL:    "http://test.ru",
				shortURL:       "/2",
				shouldRedirect: false,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()

			storage, err := fs.NewFileStorage("./test.json")
			if err != nil {
				t.Errorf("failed to initialize a new storage: %v", err)
				return
			}
			defer storage.DeleteStorageFile()
			for url := range test.args.urls {
				storage.Put(ctx, url, test.args.urls[url], "")
			}

			testApp := app.NewApp(&config.ServerConfig{}, storage)
			r := setupRouter(testApp)
			req := httptest.NewRequest(http.MethodGet, test.args.shortURL, nil)

			r.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			if test.args.shouldRedirect {
				assert.Equal(t, test.args.originalURL, res.Header.Get("Location"))
				assert.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
			} else {
				assert.Equal(t, http.StatusNotFound, res.StatusCode)
			}
		})
	}
}

func TestShortURLV1(t *testing.T) {
	type args struct {
		urls        map[string]string
		originalURL string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "add new url to empty map",
			args: args{
				urls:        make(map[string]string),
				originalURL: "https://test.ru",
			},
		},
		{
			name: "add new url to map",
			args: args{
				urls: map[string]string{
					"abc": "https://test.com",
				},
				originalURL: "https://test.ru",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()

			storage, err := fs.NewFileStorage("./test.json")
			if err != nil {
				t.Errorf("failed to initialize a new storage: %v", err)
				return
			}
			defer storage.DeleteStorageFile()
			for url := range test.args.urls {
				storage.Put(ctx, url, test.args.urls[url], "")
			}

			testApp := app.NewApp(&config.ServerConfig{}, storage)
			r := setupRouter(testApp)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte(test.args.originalURL)))
			req.Header.Add("Content-Type", "text/plain")

			r.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)

			require.NoError(t, err)
			assert.NotEmpty(t, body)
		})
	}
}

func TestShortURLV2(t *testing.T) {
	type args struct {
		urls        map[string]string
		originalURL string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "add new url to empty map",
			args: args{
				urls:        make(map[string]string),
				originalURL: "https://test.ru",
			},
		},
		{
			name: "add new url to map",
			args: args{
				urls: map[string]string{
					"abc": "https://test.com",
				},
				originalURL: "https://test.ru",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()

			storage, err := fs.NewFileStorage("./test.json")
			if err != nil {
				t.Errorf("failed to initialize a new storage: %v", err)
				return
			}
			defer storage.DeleteStorageFile()
			for url := range tt.args.urls {
				storage.Put(ctx, url, tt.args.urls[url], "")
			}

			testApp := app.NewApp(&config.ServerConfig{}, storage)
			r := setupRouter(testApp)
			reqObj := models.ShortenReq{
				URL: tt.args.originalURL,
			}
			obj, err := json.Marshal(reqObj)
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(obj))
			req.Header.Add("Content-Type", "application/json")

			r.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)

			require.NoError(t, err)
			assert.NotEmpty(t, body)
		})
	}
}

func BenchmarkShortUrl(b *testing.B) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	length := 10

	storage, err := fs.NewFileStorage("./test.json")
	if err != nil {
		b.Errorf("failed to initialize a new storage: %v", err)
		return
	}
	defer storage.DeleteStorageFile()

	testApp := app.NewApp(&config.ServerConfig{}, storage)
	r := setupRouter(testApp)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		randURL, _ := utils.GenerateRandomString(length)
		randURL = fmt.Sprintf("%s.ru", randURL)
		reqObj := models.ShortenReq{
			URL: randURL,
		}
		obj, _ := json.Marshal(reqObj)
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(obj))
		req.Header.Add("Content-Type", "application/json")
		b.StartTimer()

		r.ServeHTTP(w, req)

		w.Result()
	}
}
