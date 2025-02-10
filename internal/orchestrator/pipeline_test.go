package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sleepstars/deepempower/internal/config"
	"github.com/sleepstars/deepempower/internal/logger"
	"github.com/sleepstars/deepempower/internal/mocks"
	"github.com/sleepstars/deepempower/internal/modelbridge"
	"github.com/sleepstars/deepempower/internal/models"
	"github.com/stretchr/testify/assert"
)

func init() {
	logger.InitLogger(logger.INFO, "test")
}

func TestHybridPipeline_Execute(t *testing.T) {
	// Create mock clients
	mockNormalClient := &mocks.MockModelClient{
		CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
			return &models.ChatCompletionResponse{
				Choices: []models.ChatCompletionChoice{
					{Message: models.ChatCompletionMessage{Content: "test response"}},
				},
			}, nil
		},
	}

	mockReasonerClient := &mocks.MockModelClient{
		CompleteStreamFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
			ch := make(chan *models.ChatCompletionResponse)
			go func() {
				defer close(ch)
				ch <- &models.ChatCompletionResponse{
					Choices: []models.ChatCompletionChoice{
						{Message: models.ChatCompletionMessage{
							Content:          "reasoning step",
							ReasoningContent: []string{"step 1"},
						}},
					},
				}
			}()
			return ch, nil
		},
	}

	// Create test config
	cfg := &config.PipelineConfig{
		Models: config.ModelsConfig{
			Normal: config.ModelConfig{
				APIBase: "http://test-normal",
				Model:   "gpt-3.5-turbo",
			},
			Reasoner: config.ModelConfig{
				APIBase: "http://test-reasoner",
				Model:   "gpt-4",
			},
		},
		Prompts: config.PromptsConfig{
			PreProcess:  "test prompt",
			Reasoning:   "test prompt",
			PostProcess: "test prompt",
		},
	}

	// Create pipeline with mocked bridge
	bridge := &modelbridge.ModelBridge{
		NormalClient:   mockNormalClient,
		ReasonerClient: mockReasonerClient,
		Logger:         logger.GetLogger().WithComponent("test_bridge"),
	}

	pipeline := NewHybridPipeline(cfg)
	pipeline.SetBridge(bridge)

	// Test pipeline execution
	req := &models.ChatCompletionRequest{
		Messages: []models.ChatCompletionMessage{
			{Role: "user", Content: "test input"},
		},
	}

	resp, err := pipeline.Execute(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "test response", resp.Choices[0].Message.Content)
}

func TestHybridPipeline_ExecuteErrors(t *testing.T) {
	testCases := []struct {
		name           string
		normalClient   *mocks.MockModelClient
		reasonerClient *mocks.MockModelClient
		expectErr      string
	}{
		{
			name: "Normal client error",
			normalClient: &mocks.MockModelClient{
				CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
					return nil, fmt.Errorf("normal client error")
				},
			},
			reasonerClient: &mocks.MockModelClient{},
			expectErr:      "stage normal_preprocessor failed: model call: normal client error",
		},
		{
			name: "Reasoner client error",
			normalClient: &mocks.MockModelClient{
				CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
					return &models.ChatCompletionResponse{
						Choices: []models.ChatCompletionChoice{
							{Message: models.ChatCompletionMessage{Content: "test"}},
						},
					}, nil
				},
			},
			reasonerClient: &mocks.MockModelClient{
				CompleteStreamFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
					return nil, fmt.Errorf("reasoner client error")
				},
			},
			expectErr: "stage reasoner_engine failed: model call: reasoner client error",
		},
		{
			name: "Context timeout",
			normalClient: &mocks.MockModelClient{
				CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
					<-ctx.Done()
					return nil, ctx.Err()
				},
			},
			reasonerClient: &mocks.MockModelClient{},
			expectErr:      "context deadline exceeded",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test config
			cfg := &config.PipelineConfig{
				Models: config.ModelsConfig{
					Normal: config.ModelConfig{
						APIBase: "mock://normal",
						Model:   "gpt-3.5-turbo",
					},
					Reasoner: config.ModelConfig{
						APIBase: "mock://reasoner",
						Model:   "gpt-4",
					},
				},
				Prompts: config.PromptsConfig{
					PreProcess:  "test prompt",
					Reasoning:   "test prompt",
					PostProcess: "test prompt",
				},
			}

			// Create pipeline with mocked bridge
			bridge := &modelbridge.ModelBridge{
				NormalClient:   tc.normalClient,
				ReasonerClient: tc.reasonerClient,
				Logger:         logger.GetLogger().WithComponent("test_bridge"),
			}

			pipeline := NewHybridPipeline(cfg)
			pipeline.SetBridge(bridge)

			// Test pipeline execution with timeout context
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			req := &models.ChatCompletionRequest{
				Messages: []models.ChatCompletionMessage{
					{Role: "user", Content: "test input"},
				},
			}

			_, err := pipeline.Execute(ctx, req)
			if tc.expectErr != "" {
				assert.ErrorContains(t, err, tc.expectErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
