package handlers

import (
	"context"
	"errors"
	"github.com/EvgeniyBudaev/shortener/internal/config"
	"github.com/EvgeniyBudaev/shortener/internal/logic"
	"github.com/EvgeniyBudaev/shortener/internal/models"
	"github.com/EvgeniyBudaev/shortener/internal/store/fs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"

	pb "github.com/EvgeniyBudaev/shortener/internal/handlers/proto"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestCreateShortURL(t *testing.T) {
	mockLogger, _ := zap.NewDevelopment()
	testCases := []struct {
		name           string
		input          *pb.CreateShortURLRequest
		mockResp       string
		mockErr        error
		expectedOutput string
		expectedErr    error
	}{
		{
			name: "Successful URL shortening",
			input: &pb.CreateShortURLRequest{
				UserId: "1",
				Url:    "http://example.com",
			},
			mockResp:       "http://ex.am/1",
			expectedOutput: "http://ex.am/1",
		},
		{
			name: "Error in URL shortening",
			input: &pb.CreateShortURLRequest{
				UserId: "1",
				Url:    "http://example.com",
			},
			mockErr:     errors.New("error shortening URL"),
			expectedErr: status.Errorf(codes.Internal, "error shortening URL"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage, err := fs.NewFileStorage("./test.json")
			if err != nil {
				t.Errorf("failed to initialize a new storage: %v", err)
				return
			}
			defer storage.DeleteStorageFile()
			coreLogic := logic.NewCoreLogic(&config.ServerConfig{}, storage, zap.L().Sugar())
			service := &GRPCService{logger: mockLogger.Sugar(), coreLogic: coreLogic}

			resp, err := service.CreateShortURL(context.Background(), tc.input)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedOutput, resp.Result)
			} else {
				assert.Equal(t, tc.expectedErr, err)
			}
		})
	}
}

func TestBatchCreateShortURL(t *testing.T) {
	mockLogger, _ := zap.NewDevelopment()
	testCases := []struct {
		name         string
		request      *pb.BatchCreateShortURLRequest
		mockResponse []models.URLBatchRes
		mockErr      error
		expectedCode codes.Code
	}{
		{
			name:         "Successful batch creation",
			request:      &pb.BatchCreateShortURLRequest{ /* Заполните запрос */ },
			mockResponse: []models.URLBatchRes{{ShortURL: "http://bit.ly/test", CorrelationID: "corr123"}},
			expectedCode: codes.OK,
		},
		{
			name:         "Error from core logic",
			request:      &pb.BatchCreateShortURLRequest{ /* Заполните запрос */ },
			mockErr:      errors.New("some error"),
			expectedCode: codes.Internal,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage, err := fs.NewFileStorage("./test.json")
			if err != nil {
				t.Errorf("failed to initialize a new storage: %v", err)
				return
			}
			defer storage.DeleteStorageFile()
			coreLogic := logic.NewCoreLogic(&config.ServerConfig{}, storage, zap.L().Sugar())
			service := &GRPCService{logger: mockLogger.Sugar(), coreLogic: coreLogic}
			ctx := context.Background()
			resp, err := service.BatchCreateShortURL(ctx, tc.request)
			if tc.expectedCode == codes.OK {
				assert.NoError(t, err)
				assert.Len(t, resp.Records, len(tc.mockResponse))
				assert.Equal(t, tc.mockResponse[0].ShortURL, resp.Records[0].ShortUrl)
				assert.Equal(t, tc.mockResponse[0].CorrelationID, resp.Records[0].CorrelationId)
			} else {
				assert.Error(t, err)
				st, _ := status.FromError(err)
				assert.Equal(t, tc.expectedCode, st.Code())
			}
		})
	}
}

func TestGetByShort(t *testing.T) {
	mockLogger, _ := zap.NewDevelopment()
	testCases := []struct {
		name         string
		request      *pb.GetOriginalURLRequest
		mockResponse string
		mockErr      error
		expectedCode codes.Code
	}{
		{
			name:         "Successful retrieval",
			request:      &pb.GetOriginalURLRequest{Url: "http://bit.ly/test"},
			mockResponse: "http://example.com",
			expectedCode: codes.OK,
		},
		{
			name:         "Error from core logic",
			request:      &pb.GetOriginalURLRequest{Url: "http://bit.ly/test"},
			mockErr:      errors.New("some error"),
			expectedCode: codes.Internal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage, err := fs.NewFileStorage("./test.json")
			if err != nil {
				t.Errorf("failed to initialize a new storage: %v", err)
				return
			}
			defer storage.DeleteStorageFile()
			coreLogic := logic.NewCoreLogic(&config.ServerConfig{}, storage, zap.L().Sugar())
			service := &GRPCService{logger: mockLogger.Sugar(), coreLogic: coreLogic}
			ctx := context.Background()
			resp, err := service.GetByShort(ctx, tc.request)

			if tc.expectedCode == codes.OK {
				assert.NoError(t, err)
				assert.Equal(t, tc.mockResponse, resp.OriginalUrl)
			} else {
				assert.Error(t, err)
				st, _ := status.FromError(err)
				assert.Equal(t, tc.expectedCode, st.Code())
			}
		})
	}
}

func TestGetUserURLs(t *testing.T) {
	mockLogger, _ := zap.NewDevelopment()
	testCases := []struct {
		name         string
		request      *pb.GetUserURLsRequest
		mockResponse []models.URLRecord
		mockErr      error
		expectedCode codes.Code
	}{
		{
			name:         "Успешное получение списка URL пользователя",
			request:      &pb.GetUserURLsRequest{UserId: "user123"},
			mockResponse: []models.URLRecord{{ShortURL: "http://bit.ly/test1", OriginalURL: "http://example.com/test1"}},
			expectedCode: codes.OK,
		},
		{
			name:         "Ошибка при получении списка URL пользователя",
			request:      &pb.GetUserURLsRequest{UserId: "user123"},
			mockErr:      errors.New("ошибка получения данных"),
			expectedCode: codes.Internal,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage, err := fs.NewFileStorage("./test.json")
			if err != nil {
				t.Errorf("failed to initialize a new storage: %v", err)
				return
			}
			defer storage.DeleteStorageFile()
			coreLogic := logic.NewCoreLogic(&config.ServerConfig{}, storage, zap.L().Sugar())
			service := &GRPCService{logger: mockLogger.Sugar(), coreLogic: coreLogic}
			ctx := context.Background()
			resp, err := service.GetUserURLs(ctx, tc.request)
			if tc.expectedCode == codes.OK {
				assert.NoError(t, err)
				assert.Len(t, resp.Records, len(tc.mockResponse))
				assert.Equal(t, tc.mockResponse[0].ShortURL, resp.Records[0].ShortUrl)
				assert.Equal(t, tc.mockResponse[0].OriginalURL, resp.Records[0].OriginalUrl)
			} else {
				assert.Error(t, err)
				st, _ := status.FromError(err)
				assert.Equal(t, tc.expectedCode, st.Code())
			}
		})
	}
}
