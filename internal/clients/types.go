package clients

import (
	"context"

	"github.com/codeium/deepempower/internal/models"
)

// ModelClient defines the interface for model API clients
type ModelClient interface {
	// Complete sends a completion request to the model
	Complete(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error)

	// CompleteStream sends a streaming completion request
	CompleteStream(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error)
}

// ModelClientConfig contains configuration for model clients
type ModelClientConfig struct {
	APIBase        string
	Model          string // 添加Model字段用于指定模型名称
	DisabledParams []string
	DefaultParams  map[string]interface{}
}
