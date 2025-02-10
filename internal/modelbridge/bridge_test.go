package modelbridge

import (
	"context"
	"testing"
	"time"

	"sync"

	"github.com/sleepstars/deepempower/internal/logger"
	"github.com/sleepstars/deepempower/internal/mocks"
	"github.com/sleepstars/deepempower/internal/models"
	"github.com/stretchr/testify/assert"
)

func init() {
	logger.InitLogger(logger.INFO, "test")
}

func TestModelBridge_CallNormal(t *testing.T) {
	mockClient := &mocks.MockModelClient{
		CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
			assert.Equal(t, "gpt-3.5-turbo", req.Model)
			return &models.ChatCompletionResponse{
				Choices: []models.ChatCompletionChoice{
					{Message: models.ChatCompletionMessage{Content: "test response"}},
				},
			}, nil
		},
	}

	bridge := &ModelBridge{
		NormalClient: mockClient,
		Logger:       logger.GetLogger().WithComponent("test_bridge"),
	}

	resp, err := bridge.CallNormal(context.Background(), &models.ChatCompletionRequest{
		Messages: []models.ChatCompletionMessage{
			{Role: "user", Content: "test"},
		},
		Model: "gpt-3.5-turbo",
	})

	assert.NoError(t, err)
	assert.Equal(t, "test response", resp.Choices[0].Message.Content)
}

func TestModelBridge_CallReasonerStream(t *testing.T) {
	responses := []*models.ChatCompletionResponse{
		{
			Choices: []models.ChatCompletionChoice{
				{Message: models.ChatCompletionMessage{Content: "step 1"}},
			},
		},
		{
			Choices: []models.ChatCompletionChoice{
				{Message: models.ChatCompletionMessage{ReasoningContent: []string{"reasoning 1"}}},
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
					select {
					case <-ctx.Done():
						return
					case ch <- resp:
					}
					time.Sleep(10 * time.Millisecond)
				}
			}()
			return ch, nil
		},
	}

	bridge := &ModelBridge{
		ReasonerClient: mockClient,
		Logger:         logger.GetLogger().WithComponent("test_bridge"),
	}

	respChan, err := bridge.CallReasonerStream(context.Background(), &models.ChatCompletionRequest{
		Messages: []models.ChatCompletionMessage{
			{Role: "user", Content: "test"},
		},
		Model: "gpt-4",
	})

	assert.NoError(t, err)

	var receivedResponses []*models.ChatCompletionResponse
	for resp := range respChan {
		receivedResponses = append(receivedResponses, resp)
	}

	assert.Equal(t, len(responses), len(receivedResponses))
	assert.Equal(t, "step 1", receivedResponses[0].Choices[0].Message.Content)
	assert.Equal(t, []string{"reasoning 1"}, receivedResponses[1].Choices[0].Message.ReasoningContent)
}

func TestModelBridge_ConcurrentCalls(t *testing.T) {
	// Test concurrent access to bridge methods
	numCalls := 10
	mockClient := &mocks.MockModelClient{
		CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
			time.Sleep(10 * time.Millisecond) // Simulate work
			return &models.ChatCompletionResponse{
				Choices: []models.ChatCompletionChoice{
					{Message: models.ChatCompletionMessage{Content: "test response"}},
				},
			}, nil
		},
		CompleteStreamFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
			ch := make(chan *models.ChatCompletionResponse)
			go func() {
				defer close(ch)
				time.Sleep(10 * time.Millisecond) // Simulate work
				ch <- &models.ChatCompletionResponse{
					Choices: []models.ChatCompletionChoice{
						{Message: models.ChatCompletionMessage{Content: "stream response"}},
					},
				}
			}()
			return ch, nil
		},
	}

	bridge := &ModelBridge{
		NormalClient:   mockClient,
		ReasonerClient: mockClient,
		Logger:         logger.GetLogger().WithComponent("test_bridge"),
	}

	// Test concurrent normal calls
	var wg sync.WaitGroup
	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := bridge.CallNormal(context.Background(), &models.ChatCompletionRequest{
				Model: "test",
			})
			assert.NoError(t, err)
			assert.NotNil(t, resp)
		}()
	}

	// Test concurrent reasoner calls
	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			respCh, err := bridge.CallReasonerStream(context.Background(), &models.ChatCompletionRequest{
				Model: "test",
			})
			assert.NoError(t, err)
			for range respCh {
				// Consume responses
			}
		}()
	}

	wg.Wait()
}

func TestModelBridge_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name      string
		client    *mocks.MockModelClient
		callType  string
		expectErr string
	}{
		{
			name: "Normal client panic",
			client: &mocks.MockModelClient{
				CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
					panic("unexpected panic")
				},
			},
			callType:  "normal",
			expectErr: "runtime error",
		},
		{
			name: "Reasoner stream empty response",
			client: &mocks.MockModelClient{
				CompleteStreamFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
					ch := make(chan *models.ChatCompletionResponse)
					go func() {
						defer close(ch)
						ch <- &models.ChatCompletionResponse{
							Choices: []models.ChatCompletionChoice{}, // Empty choices
						}
					}()
					return ch, nil
				},
			},
			callType: "reasoner",
		},
		{
			name: "Context cancellation",
			client: &mocks.MockModelClient{
				CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
					<-ctx.Done()
					return nil, ctx.Err()
				},
			},
			callType:  "normal",
			expectErr: "context canceled",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bridge := &ModelBridge{
				NormalClient:   tc.client,
				ReasonerClient: tc.client,
				Logger:         logger.GetLogger().WithComponent("test_bridge"),
			}

			ctx, cancel := context.WithCancel(context.Background())
			if tc.expectErr == "context canceled" {
				cancel()
			} else {
				defer cancel()
			}

			if tc.callType == "normal" {
				_, err := bridge.CallNormal(ctx, &models.ChatCompletionRequest{Model: "test"})
				if tc.expectErr != "" {
					assert.ErrorContains(t, err, tc.expectErr)
				} else {
					assert.NoError(t, err)
				}
			} else {
				respCh, err := bridge.CallReasonerStream(ctx, &models.ChatCompletionRequest{Model: "test"})
				if tc.expectErr != "" {
					assert.ErrorContains(t, err, tc.expectErr)
				} else {
					assert.NoError(t, err)
					for resp := range respCh {
						if len(resp.Choices) == 0 {
							// Test passes - we expect empty choices in this case
							return
						}
					}
				}
			}
		})
	}
}

func TestModelBridge_StreamFilterEmptyResponses(t *testing.T) {
	responses := []*models.ChatCompletionResponse{
		{
			Choices: []models.ChatCompletionChoice{}, // Empty
		},
		{
			Choices: []models.ChatCompletionChoice{
				{Message: models.ChatCompletionMessage{}}, // Empty content
			},
		},
		{
			Choices: []models.ChatCompletionChoice{
				{Message: models.ChatCompletionMessage{
					Content: "valid content",
				}},
			},
		},
	}

	mockClient := &mocks.MockModelClient{
		CompleteStreamFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
			ch := make(chan *models.ChatCompletionResponse)
			go func() {
				defer close(ch)
				for _, resp := range responses {
					select {
					case <-ctx.Done():
						return
					case ch <- resp:
					}
				}
			}()
			return ch, nil
		},
	}

	bridge := &ModelBridge{
		ReasonerClient: mockClient,
		Logger:         logger.GetLogger().WithComponent("test_bridge"),
	}

	respCh, err := bridge.CallReasonerStream(context.Background(), &models.ChatCompletionRequest{Model: "test"})
	assert.NoError(t, err)

	var validResponses []*models.ChatCompletionResponse
	for resp := range respCh {
		validResponses = append(validResponses, resp)
	}

	// Only the response with valid content should make it through
	assert.Equal(t, 1, len(validResponses))
	assert.Equal(t, "valid content", validResponses[0].Choices[0].Message.Content)
}
