package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/codeium/deepempower/internal/config"
	"github.com/codeium/deepempower/internal/logger"
	"github.com/codeium/deepempower/internal/mocks"
	"github.com/codeium/deepempower/internal/modelbridge"
	"github.com/codeium/deepempower/internal/models"
	"github.com/codeium/deepempower/internal/orchestrator"
	"github.com/stretchr/testify/assert"
)

func init() {
	logger.InitLogger(logger.INFO, "integration_test")
}

func TestPipelineIntegration(t *testing.T) {
	// Create mock clients with model validation
	mockNormal := &mocks.MockModelClient{
		CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
			assert.Equal(t, "gpt-3.5-turbo", req.Model)
			return &models.ChatCompletionResponse{
				Choices: []models.ChatCompletionChoice{
					{Message: models.ChatCompletionMessage{
						Content: "Mock normal response",
						Role:    "assistant",
					}},
				},
			}, nil
		},
	}

	mockReasoner := &mocks.MockModelClient{
		CompleteStreamFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
			assert.Equal(t, "gpt-4", req.Model)
			ch := make(chan *models.ChatCompletionResponse)
			go func() {
				defer close(ch)
				ch <- &models.ChatCompletionResponse{
					Choices: []models.ChatCompletionChoice{
						{Message: models.ChatCompletionMessage{
							Content:          "Mock reasoning response",
							ReasoningContent: []string{"Step 1", "Step 2"},
							Role:             "assistant",
						}},
					},
				}
			}()
			return ch, nil
		},
	}

	// Load test configuration
	cfg := &config.PipelineConfig{
		Models: config.ModelsConfig{
			Normal: config.ModelConfig{
				APIBase: "mock://normal",
				Model:   "gpt-3.5-turbo",
				DefaultParams: map[string]interface{}{
					"temperature": 0.7,
					"max_tokens":  1000,
				},
			},
			Reasoner: config.ModelConfig{
				APIBase: "mock://reasoner",
				Model:   "gpt-4",
				DisabledParams: []string{
					"temperature",
					"presence_penalty",
					"frequency_penalty",
				},
			},
		},
		Prompts: config.PromptsConfig{
			PreProcess:  "Test preprocessing prompt",
			Reasoning:   "Test reasoning prompt",
			PostProcess: "Test postprocessing prompt",
		},
	}

	// Create pipeline with mock bridge
	bridge := &modelbridge.ModelBridge{
		NormalClient:   mockNormal,
		ReasonerClient: mockReasoner,
		Logger:         logger.GetLogger().WithComponent("test_bridge"),
	}

	pipeline := orchestrator.NewHybridPipeline(cfg)
	pipeline.SetBridge(bridge)

	// Test cases
	testCases := []struct {
		name     string
		input    string
		timeout  time.Duration
		validate func(t *testing.T, resp *models.ChatCompletionResponse, err error)
	}{
		{
			name:    "Simple question",
			input:   "What is 2+2?",
			timeout: 2 * time.Second,
			validate: func(t *testing.T, resp *models.ChatCompletionResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Choices)
				assert.NotEmpty(t, resp.Choices[0].Message.Content)
			},
		},
		{
			name:    "Complex question",
			input:   "Explain how a car engine works",
			timeout: 2 * time.Second,
			validate: func(t *testing.T, resp *models.ChatCompletionResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Choices)
				assert.NotEmpty(t, resp.Choices[0].Message.Content)
				assert.NotEmpty(t, resp.Choices[0].Message.ReasoningContent)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			req := &models.ChatCompletionRequest{
				Messages: []models.ChatCompletionMessage{
					{Role: "user", Content: tc.input},
				},
			}

			resp, err := pipeline.Execute(ctx, req)
			tc.validate(t, resp, err)
		})
	}
}

func TestPipelineIntegrationErrorRecovery(t *testing.T) {
	// Test error recovery scenarios
	testCases := []struct {
		name     string
		setup    func() (*mocks.MockModelClient, *mocks.MockModelClient)
		input    string
		validate func(t *testing.T, resp *models.ChatCompletionResponse, err error)
	}{
		{
			name: "Recover from temporary failure",
			setup: func() (*mocks.MockModelClient, *mocks.MockModelClient) {
				attemptCount := 0
				mockNormal := &mocks.MockModelClient{
					CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
						attemptCount++
						if attemptCount == 1 {
							// Fail first attempt
							return nil, fmt.Errorf("temporary error")
						}
						return &models.ChatCompletionResponse{
							Choices: []models.ChatCompletionChoice{
								{Message: models.ChatCompletionMessage{Content: "recovered response"}},
							},
						}, nil
					},
				}
				mockReasoner := &mocks.MockModelClient{
					CompleteStreamFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
						ch := make(chan *models.ChatCompletionResponse)
						go func() {
							defer close(ch)
							ch <- &models.ChatCompletionResponse{
								Choices: []models.ChatCompletionChoice{
									{Message: models.ChatCompletionMessage{
										Content:          "reasoning response",
										ReasoningContent: []string{"recovery step"},
									}},
								},
							}
						}()
						return ch, nil
					},
				}
				return mockNormal, mockReasoner
			},
			input: "test with recovery",
			validate: func(t *testing.T, resp *models.ChatCompletionResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, "recovered response", resp.Choices[0].Message.Content)
			},
		},
		{
			name: "Empty reasoning chain handling",
			setup: func() (*mocks.MockModelClient, *mocks.MockModelClient) {
				mockNormal := &mocks.MockModelClient{
					CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
						return &models.ChatCompletionResponse{
							Choices: []models.ChatCompletionChoice{
								{Message: models.ChatCompletionMessage{Content: "normal response"}},
							},
						}, nil
					},
				}
				mockReasoner := &mocks.MockModelClient{
					CompleteStreamFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
						ch := make(chan *models.ChatCompletionResponse)
						go func() {
							defer close(ch)
							// Send responses with no reasoning content
							ch <- &models.ChatCompletionResponse{
								Choices: []models.ChatCompletionChoice{
									{Message: models.ChatCompletionMessage{
										Content: "thinking...",
									}},
								},
							}
						}()
						return ch, nil
					},
				}
				return mockNormal, mockReasoner
			},
			input: "test empty reasoning",
			validate: func(t *testing.T, resp *models.ChatCompletionResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Empty(t, resp.Choices[0].Message.ReasoningContent)
			},
		},
		{
			name: "Large message handling",
			setup: func() (*mocks.MockModelClient, *mocks.MockModelClient) {
				mockNormal := &mocks.MockModelClient{
					CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
						return &models.ChatCompletionResponse{
							Choices: []models.ChatCompletionChoice{
								{Message: models.ChatCompletionMessage{
									Content: strings.Repeat("large ", 1000),
								}},
							},
						}, nil
					},
				}
				mockReasoner := &mocks.MockModelClient{
					CompleteStreamFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
						ch := make(chan *models.ChatCompletionResponse)
						go func() {
							defer close(ch)
							// Send large streaming response
							for i := 0; i < 5; i++ {
								ch <- &models.ChatCompletionResponse{
									Choices: []models.ChatCompletionChoice{
										{Message: models.ChatCompletionMessage{
											Content:          fmt.Sprintf("part %d", i),
											ReasoningContent: []string{strings.Repeat("step ", 100)},
										}},
									},
								}
							}
						}()
						return ch, nil
					},
				}
				return mockNormal, mockReasoner
			},
			input: "test large message",
			validate: func(t *testing.T, resp *models.ChatCompletionResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Greater(t, len(resp.Choices[0].Message.Content), 5000)
				assert.NotEmpty(t, resp.Choices[0].Message.ReasoningContent)
			},
		},
		{
			name: "Context cancellation handling",
			setup: func() (*mocks.MockModelClient, *mocks.MockModelClient) {
				mockNormal := &mocks.MockModelClient{
					CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
						return &models.ChatCompletionResponse{
							Choices: []models.ChatCompletionChoice{
								{Message: models.ChatCompletionMessage{Content: "initial"}},
							},
						}, nil
					},
				}
				mockReasoner := &mocks.MockModelClient{
					CompleteStreamFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
						ch := make(chan *models.ChatCompletionResponse)
						go func() {
							defer close(ch)
							// Simulate long processing that should be cancelled
							select {
							case <-ctx.Done():
								return
							case <-time.After(500 * time.Millisecond):
								ch <- &models.ChatCompletionResponse{
									Choices: []models.ChatCompletionChoice{
										{Message: models.ChatCompletionMessage{Content: "too late"}},
									},
								}
							}
						}()
						return ch, nil
					},
				}
				return mockNormal, mockReasoner
			},
			input: "test cancellation",
			validate: func(t *testing.T, resp *models.ChatCompletionResponse, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "context deadline exceeded")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockNormal, mockReasoner := tc.setup()

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
					PreProcess:  "Test preprocessing prompt",
					Reasoning:   "Test reasoning prompt",
					PostProcess: "Test postprocessing prompt",
				},
			}

			// Create pipeline with mock clients
			bridge := &modelbridge.ModelBridge{
				NormalClient:   mockNormal,
				ReasonerClient: mockReasoner,
				Logger:         logger.GetLogger().WithComponent("test_bridge"),
			}

			pipeline := orchestrator.NewHybridPipeline(cfg)
			pipeline.SetBridge(bridge)

			// Set short timeout for cancellation test
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			req := &models.ChatCompletionRequest{
				Messages: []models.ChatCompletionMessage{
					{Role: "user", Content: tc.input},
				},
			}

			resp, err := pipeline.Execute(ctx, req)
			tc.validate(t, resp, err)
		})
	}
}
