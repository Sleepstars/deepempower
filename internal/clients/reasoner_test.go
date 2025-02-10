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

func TestReasonerClient_Complete(t *testing.T) {
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
			name: "successful request with disabled params",
			config: ModelClientConfig{
				APIBase:        "test-server",
				Model:          "test-model",
				DisabledParams: []string{"temperature"},
			},
			request: &models.ChatCompletionRequest{
				Model: "test-model",
				Messages: []models.ChatCompletionMessage{
					{Role: "user", Content: "test message"},
				},
				Temperature: 0.7,
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

				// Verify request body and parameter filtering
				var reqMap map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqMap)
				require.NoError(t, err)

				// Verify disabled parameters are removed
				for _, param := range tc.config.DisabledParams {
					_, hasParam := reqMap[param]
					assert.False(t, hasParam, "disabled parameter %s should not be present", param)
				}

				// Send response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tc.response)
			}))
			defer server.Close()

			// Update config with test server URL
			tc.config.APIBase = server.URL

			// Create client
			client := NewReasonerClient(tc.config)

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

func TestReasonerClient_CompleteStream(t *testing.T) {
	tests := []struct {
		name        string
		config      ModelClientConfig
		request     *models.ChatCompletionRequest
		responses   []openai.ChatCompletionStreamResponse
		expectedErr string
		expected    struct {
			contents   []string
			reasonings []string
		}
	}{
		{
			name: "successful stream with reasoning",
			config: ModelClientConfig{
				APIBase:        "test-server",
				Model:          "test-model",
				DisabledParams: []string{"temperature"},
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
								Content: "thinking...",
							},
						},
					},
				},
				{
					Choices: []openai.ChatCompletionStreamChoice{
						{
							Delta: openai.ChatCompletionStreamChoiceDelta{
								Content: "final answer",
							},
							FinishReason: openai.FinishReasonStop,
						},
					},
				},
			},
			expected: struct {
				contents   []string
				reasonings []string
			}{
				contents:   []string{"thinking...", "final answer"},
				reasonings: []string{},
			},
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

				// Verify request body and parameter filtering
				var reqMap map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqMap)
				require.NoError(t, err)

				// Verify disabled parameters are removed
				for _, param := range tc.config.DisabledParams {
					_, hasParam := reqMap[param]
					assert.False(t, hasParam, "disabled parameter %s should not be present", param)
				}

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
			client := NewReasonerClient(tc.config)

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

			contents := make([]string, 0)
			reasonings := make([]string, 0)
			for resp := range respChan {
				contents = append(contents, resp.Choices[0].Message.Content)
				reasonings = append(reasonings, resp.Choices[0].Message.ReasoningContent...)
			}

			assert.Equal(t, tc.expected.contents, contents)
			assert.Equal(t, tc.expected.reasonings, reasonings)
		})
	}
}
