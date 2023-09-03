package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortUrl(t *testing.T) {
	type args struct {
		urls        map[string][]byte
		originalURL string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "add new url to empty map",
			args: args{
				urls:        map[string][]byte{},
				originalURL: "https://test.ru",
			},
		},
		{
			name: "add new url to map",
			args: args{
				urls: map[string][]byte{
					"abc": []byte("https://test.ru"),
				},
				originalURL: "https://test.ru",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := UrlHandler(test.args.urls)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte(test.args.originalURL)))

			w := httptest.NewRecorder()
			handler(w, req)

			res := w.Result()
			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)

			require.NoError(t, err)
			assert.NotEmpty(t, body)
		})
	}
}

func TestRedirectUrl(t *testing.T) {
	type args struct {
		urls           map[string][]byte
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
				urls: map[string][]byte{
					"1": []byte("http://test.ru"),
				},
				originalURL:    "http://test.ru",
				shortURL:       "1",
				shouldRedirect: true,
			},
		},
		{
			name: "error short url not found",
			args: args{
				urls: map[string][]byte{
					"1": []byte("http://test.ru"),
				},
				originalURL:    "http://test.ru",
				shortURL:       "2",
				shouldRedirect: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := UrlHandler(tt.args.urls)
			req := httptest.NewRequest(http.MethodGet, "/"+tt.args.shortURL, nil)

			w := httptest.NewRecorder()
			handler(w, req)

			res := w.Result()
			defer res.Body.Close()
			_, err := io.ReadAll(res.Body)

			require.NoError(t, err)
			if tt.args.shouldRedirect {
				assert.Equal(t, res.Header.Get("Location"), tt.args.originalURL)
				assert.Equal(t, res.StatusCode, http.StatusTemporaryRedirect)
			} else {
				assert.Equal(t, res.StatusCode, http.StatusNotFound)
			}
		})
	}
}
