package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sleepstars/deepempower/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNormalClient_Complete(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)

		// Decode request body
		var req models.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)

		// Verify request content
		assert.Equal(t, "test-model", req.Model)

		// Send response
		resp := models.ChatCompletionResponse{
			Choices: []models.ChatCompletionChoice{
				{
					Message: models.ChatCompletionMessage{
						Role:    "assistant",
						Content: "test response",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewNormalClient(ModelClientConfig{
		APIBase: server.URL,
		Model:   "test-model",
	})

	// Make request
	resp, err := client.Complete(context.Background(), &models.ChatCompletionRequest{
		Messages: []models.ChatCompletionMessage{
			{Role: "user", Content: "test"},
		},
	})

	// Verify response
	assert.NoError(t, err)
	assert.Equal(t, "test response", resp.Choices[0].Message.Content)
}

func TestNormalClient_CompleteStream(t *testing.T) {
	responses := []models.ChatCompletionResponse{
		{
			Choices: []models.ChatCompletionChoice{
				{Message: models.ChatCompletionMessage{Content: "part 1"}},
			},
		},
		{
			Choices: []models.ChatCompletionChoice{
				{Message: models.ChatCompletionMessage{Content: "part 2"}},
			},
		},
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify streaming headers
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))

		// Send streaming responses
		for _, resp := range responses {
			json.NewEncoder(w).Encode(resp)
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewNormalClient(ModelClientConfig{
		APIBase: server.URL,
		Model:   "test-model",
	})

	// Make streaming request
	respChan, err := client.CompleteStream(context.Background(), &models.ChatCompletionRequest{
		Messages: []models.ChatCompletionMessage{
			{Role: "user", Content: "test"},
		},
		Stream: true,
	})

	// Verify response stream
	assert.NoError(t, err)

	var receivedResponses []string
	for resp := range respChan {
		receivedResponses = append(receivedResponses, resp.Choices[0].Message.Content)
	}

	assert.Equal(t, []string{"part 1", "part 2"}, receivedResponses)
}
