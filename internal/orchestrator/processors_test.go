package orchestrator

import (
	"context"
	"testing"

	"github.com/codeium/deepempower/internal/logger"
	"github.com/codeium/deepempower/internal/mocks"
	"github.com/codeium/deepempower/internal/modelbridge"
	"github.com/codeium/deepempower/internal/models"
	"github.com/stretchr/testify/assert"
)

func init() {
	logger.InitLogger(logger.INFO, "test")
}

func TestNormalPreprocessor_Execute(t *testing.T) {
	mockClient := &mocks.MockModelClient{
		CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
			assert.Equal(t, "gpt-3.5-turbo", req.Model)
			return &models.ChatCompletionResponse{
				Choices: []models.ChatCompletionChoice{
					{Message: models.ChatCompletionMessage{Content: "preprocessed"}},
				},
			}, nil
		},
	}

	bridge := &modelbridge.ModelBridge{
		NormalClient: mockClient,
		Logger:       logger.GetLogger().WithComponent("test_bridge"),
	}

	processor := newNormalPreprocessor("template ${input}", bridge)
	payload := &Payload{
		OriginalRequest: &models.ChatCompletionRequest{
			Model: "gpt-3.5-turbo",
			Messages: []models.ChatCompletionMessage{
				{Role: "user", Content: "test"},
			},
		},
	}

	err := processor.Execute(context.Background(), payload)
	assert.NoError(t, err)
	assert.Equal(t, "preprocessed", payload.IntermContent)
}

func TestReasonerEngine_Execute(t *testing.T) {
	responses := []*models.ChatCompletionResponse{
		{
			Choices: []models.ChatCompletionChoice{
				{Message: models.ChatCompletionMessage{
					Content:          "step 1",
					ReasoningContent: []string{"reasoning 1"},
				}},
			},
		},
		{
			Choices: []models.ChatCompletionChoice{
				{Message: models.ChatCompletionMessage{
					Content:          "step 2",
					ReasoningContent: []string{"reasoning 2"},
				}},
			},
		},
	}

	mockClient := &mocks.MockModelClient{
		CompleteStreamFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
			assert.Equal(t, "gpt-4", req.Model)
			assert.True(t, req.Stream)

			ch := make(chan *models.ChatCompletionResponse)
			go func() {
				defer close(ch)
				for _, resp := range responses {
					ch <- resp
				}
			}()
			return ch, nil
		},
	}

	bridge := &modelbridge.ModelBridge{
		ReasonerClient: mockClient,
		Logger:         logger.GetLogger().WithComponent("test_bridge"),
	}

	processor := newReasonerEngine("template ${input}", bridge)
	payload := &Payload{
		OriginalRequest: &models.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []models.ChatCompletionMessage{
				{Role: "user", Content: "test"},
			},
		},
		IntermContent: "preprocessed",
	}

	err := processor.Execute(context.Background(), payload)
	assert.NoError(t, err)
	assert.Equal(t, "step 2", payload.IntermContent)
	assert.Equal(t, []string{"reasoning 1", "reasoning 2"}, payload.ReasoningChain)
}

func TestNormalPostprocessor_Execute(t *testing.T) {
	mockClient := &mocks.MockModelClient{
		CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
			assert.Equal(t, "gpt-3.5-turbo", req.Model)
			return &models.ChatCompletionResponse{
				Choices: []models.ChatCompletionChoice{
					{Message: models.ChatCompletionMessage{Content: "final response"}},
				},
			}, nil
		},
	}

	bridge := &modelbridge.ModelBridge{
		NormalClient: mockClient,
		Logger:       logger.GetLogger().WithComponent("test_bridge"),
	}

	processor := newNormalPostprocessor("template ${input}", bridge)
	payload := &Payload{
		OriginalRequest: &models.ChatCompletionRequest{
			Model: "gpt-3.5-turbo",
			Messages: []models.ChatCompletionMessage{
				{Role: "user", Content: "test"},
			},
		},
		IntermContent:  "reasoned",
		ReasoningChain: []string{"step 1", "step 2"},
	}

	err := processor.Execute(context.Background(), payload)
	assert.NoError(t, err)
	assert.Equal(t, "final response", payload.FinalContent)
}
