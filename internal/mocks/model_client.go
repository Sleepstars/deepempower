package mocks

import (
	"context"

	"github.com/codeium/deepempower/internal/models"
)

// MockModelClient implements ModelClient interface for testing
type MockModelClient struct {
	CompleteFunc       func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error)
	CompleteStreamFunc func(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error)
}

func (m *MockModelClient) Complete(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	if m.CompleteFunc != nil {
		return m.CompleteFunc(ctx, req)
	}
	return &models.ChatCompletionResponse{}, nil
}

func (m *MockModelClient) CompleteStream(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
	if m.CompleteStreamFunc != nil {
		return m.CompleteStreamFunc(ctx, req)
	}
	ch := make(chan *models.ChatCompletionResponse)
	close(ch)
	return ch, nil
}
