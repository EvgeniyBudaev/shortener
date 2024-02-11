package handlers

import (
	"context"
	"errors"
	"github.com/EvgeniyBudaev/shortener/internal/config"
	"github.com/EvgeniyBudaev/shortener/internal/logic"
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
