package modelbridge

import (
	"context"
	"testing"
	"time"

	"github.com/codeium/deepempower/internal/clients"
	"github.com/codeium/deepempower/internal/mocks"
	"github.com/codeium/deepempower/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestModelBridge_CallNormal(t *testing.T) {
	mockClient := &mocks.MockModelClient{
		CompleteFunc: func(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
			assert.Equal(t, "Normal", req.Model)
			return &models.ChatCompletionResponse{
				Choices: []models.ChatCompletionChoice{
					{Message: models.ChatCompletionMessage{Content: "test response"}},
				},
			}, nil
		},
	}

	bridge := &ModelBridge{
		normalClient: mockClient,
	}

	resp, err := bridge.CallNormal(context.Background(), &models.ChatCompletionRequest{
		Messages: []models.ChatCompletionMessage{
			{Role: "user", Content: "test"},
		},
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
			assert.Equal(t, "Reasoner", req.Model)
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
		reasonerClient: mockClient,
	}

	respChan, err := bridge.CallReasonerStream(context.Background(), &models.ChatCompletionRequest{
		Messages: []models.ChatCompletionMessage{
			{Role: "user", Content: "test"},
		},
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
