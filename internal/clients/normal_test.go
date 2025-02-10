package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sleepstars/deepempower/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalClient_Complete(t *testing.T) {
	tests := []struct {
		name           string
		config         ModelClientConfig
		request        *models.ChatCompletionRequest
		expectedReq    openai.ChatCompletionRequest
		response       openai.ChatCompletionResponse
		expectedErr    string
		expectedResult *models.ChatCompletionResponse
	}{
		{
			name: "successful request with model override",
			config: ModelClientConfig{
				APIBase: "test-server",
				Model:   "default-model",
				DefaultParams: map[string]interface{}{
					"temperature": 0.7,
					"max_tokens":  100,
				},
			},
			request: &models.ChatCompletionRequest{
				Model: "test-model",
				Messages: []models.ChatCompletionMessage{
					{Role: "user", Content: "test message"},
				},
				Temperature: 0.5,
			},
			expectedReq: openai.ChatCompletionRequest{
				Model: "test-model",
				Messages: []openai.ChatCompletionMessage{
					{Role: "user", Content: "test message"},
				},
				Temperature: 0.5,
				MaxTokens:   100,
			},
			response: openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "test response",
						},
						FinishReason: openai.FinishReasonStop,
					},
				},
			},
			expectedResult: &models.ChatCompletionResponse{
				Choices: []models.ChatCompletionChoice{
					{
						Message: models.ChatCompletionMessage{
							Role:    "assistant",
							Content: "test response",
						},
						FinishReason: "stop",
					},
				},
			},
		},
		{
			name: "empty response",
			config: ModelClientConfig{
				APIBase: "test-server",
				Model:   "test-model",
			},
			request: &models.ChatCompletionRequest{
				Messages: []models.ChatCompletionMessage{
					{Role: "user", Content: "test"},
				},
			},
			response: openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{},
			},
			expectedErr: "no choices in response",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/chat/completions", r.URL.Path)

				// Verify request body
				var req openai.ChatCompletionRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)

				// Send response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tc.response)
			}))
			defer server.Close()

			// Update config with test server URL
			tc.config.APIBase = server.URL

			// Create client
			client := NewNormalClient(tc.config)

			// Make request
			resp, err := client.Complete(context.Background(), tc.request)

			// Verify results
			if tc.expectedErr != "" {
				assert.EqualError(t, err, tc.expectedErr)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResult, resp)
			}
		})
	}
}

func TestNormalClient_CompleteStream(t *testing.T) {
	tests := []struct {
		name        string
		config      ModelClientConfig
		request     *models.ChatCompletionRequest
		responses   []openai.ChatCompletionStreamResponse
		expectedErr string
		expected    []string
	}{
		{
			name: "successful stream",
			config: ModelClientConfig{
				APIBase: "test-server",
				Model:   "test-model",
			},
			request: &models.ChatCompletionRequest{
				Messages: []models.ChatCompletionMessage{
					{Role: "user", Content: "test"},
				},
			},
			responses: []openai.ChatCompletionStreamResponse{
				{
					Choices: []openai.ChatCompletionStreamChoice{
						{
							Delta: openai.ChatCompletionStreamChoiceDelta{
								Role:    "assistant",
								Content: "part 1",
							},
						},
					},
				},
				{
					Choices: []openai.ChatCompletionStreamChoice{
						{
							Delta: openai.ChatCompletionStreamChoiceDelta{
								Content: "part 2",
							},
							FinishReason: openai.FinishReasonStop,
						},
					},
				},
			},
			expected: []string{"part 1", "part 2"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/chat/completions", r.URL.Path)
				assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))

				// Send streaming responses
				for _, resp := range tc.responses {
					data, _ := json.Marshal(resp)
					fmt.Fprintf(w, "data: %s\n\n", data)
					w.(http.Flusher).Flush()
				}
			}))
			defer server.Close()

			// Update config with test server URL
			tc.config.APIBase = server.URL

			// Create client
			client := NewNormalClient(tc.config)

			// Make streaming request
			respChan, err := client.CompleteStream(context.Background(), tc.request)

			// Verify initial error if expected
			if tc.expectedErr != "" {
				assert.EqualError(t, err, tc.expectedErr)
				assert.Nil(t, respChan)
				return
			}

			// Verify streaming responses
			assert.NoError(t, err)
			require.NotNil(t, respChan)

			var received []string
			for resp := range respChan {
				received = append(received, resp.Choices[0].Message.Content)
			}

			assert.Equal(t, tc.expected, received)
		})
	}
}
