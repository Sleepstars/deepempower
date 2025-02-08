package orchestrator

import (
	"context"
	"testing"

	"github.com/codeium/deepempower/internal/config"
	"github.com/codeium/deepempower/internal/mocks"
	"github.com/codeium/deepempower/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestHybridPipeline_Execute(t *testing.T) {
	// Setup mock responses
	normalPreprocessResponse := &models.ChatCompletionResponse{
		Choices: []models.ChatCompletionChoice{
			{Message: models.ChatCompletionMessage{Content: "preprocessed"}},
		},
	}

	reasonerResponses := []*models.ChatCompletionResponse{
		{
			Choices: []models.ChatCompletionChoice{
				{Message: models.ChatCompletionMessage{
					Content:          "reasoning step",
					ReasoningContent: []string{"step 1"},
				}},
			},
		},
	}

	normalPostprocessResponse := &models.ChatCompletionResponse{
		Choices: []models.ChatCompletionChoice{
			{Message: models.ChatCompletionMessage{Content: "final response"}},
		},
	}

	// Create mock clients
	mockNormalClient := &mocks.MockModelClient{
		CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
			if req.Messages[0].Content == "preprocess" {
				return normalPreprocessResponse, nil
			}
			return normalPostprocessResponse, nil
		},
	}

	mockReasonerClient := &mocks.MockModelClient{
		CompleteStreamFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
			ch := make(chan *models.ChatCompletionResponse)
			go func() {
				defer close(ch)
				for _, resp := range reasonerResponses {
					ch <- resp
				}
			}()
			return ch, nil
		},
	}

	// Create test config
	cfg := &config.PipelineConfig{
		Models: config.ModelsConfig{
			Normal: config.ModelConfig{
				APIBase: "http://normal",
			},
			Reasoner: config.ModelConfig{
				APIBase: "http://reasoner",
			},
		},
		Prompts: config.PromptsConfig{
			PreProcess:  "preprocess",
			Reasoning:   "reason",
			PostProcess: "postprocess",
		},
	}

	// Create pipeline
	pipeline := NewHybridPipeline(cfg)

	// Test pipeline execution
	req := &models.ChatCompletionRequest{
		Messages: []models.ChatCompletionMessage{
			{Role: "user", Content: "test input"},
		},
	}

	resp, err := pipeline.Execute(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "final response", resp.Choices[0].Message.Content)
	assert.Equal(t, []string{"step 1"}, resp.Choices[0].Message.ReasoningContent)
}
