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

func TestReasonerClient_Complete(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode request body to verify parameter filtering
		var reqMap map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&reqMap)
		assert.NoError(t, err)

		// Verify disabled parameters are removed
		_, hasTemperature := reqMap["temperature"]
		assert.False(t, hasTemperature)

		// Send response with reasoning content
		resp := struct {
			Choices []struct {
				Message struct {
					Content          string   `json:"content"`
					ReasoningContent []string `json:"reasoning_content"`
				} `json:"message"`
			} `json:"choices"`
		}{
			Choices: []struct {
				Message struct {
					Content          string   `json:"content"`
					ReasoningContent []string `json:"reasoning_content"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Content          string   `json:"content"`
						ReasoningContent []string `json:"reasoning_content"`
					}{
						Content:          "test response",
						ReasoningContent: []string{"step 1", "step 2"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client with test server URL and disabled parameters
	client := NewReasonerClient(ModelClientConfig{
		APIBase:        server.URL,
		Model:          "test-model",
		DisabledParams: []string{"temperature"},
	})

	// Make request with a parameter that should be filtered
	reqMap := map[string]interface{}{
		"temperature": 0.7,
		"model":       "test-model",
		"messages": []models.ChatCompletionMessage{
			{Role: "user", Content: "test"},
		},
	}
	reqData, _ := json.Marshal(reqMap)
	var req models.ChatCompletionRequest
	json.Unmarshal(reqData, &req)

	// Send request
	resp, err := client.Complete(context.Background(), &req)

	// Verify response
	assert.NoError(t, err)
	assert.Equal(t, "test response", resp.Choices[0].Message.Content)
	assert.Equal(t, []string{"step 1", "step 2"}, resp.Choices[0].Message.ReasoningContent)
}

func TestReasonerClient_CompleteStream(t *testing.T) {
	responses := []struct {
		Choices []struct {
			Message struct {
				Content          string   `json:"content"`
				ReasoningContent []string `json:"reasoning_content"`
			} `json:"message"`
		} `json:"choices"`
	}{
		{
			Choices: []struct {
				Message struct {
					Content          string   `json:"content"`
					ReasoningContent []string `json:"reasoning_content"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Content          string   `json:"content"`
						ReasoningContent []string `json:"reasoning_content"`
					}{
						Content:          "thinking...",
						ReasoningContent: []string{"analyzing problem"},
					},
				},
			},
		},
		{
			Choices: []struct {
				Message struct {
					Content          string   `json:"content"`
					ReasoningContent []string `json:"reasoning_content"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Content          string   `json:"content"`
						ReasoningContent []string `json:"reasoning_content"`
					}{
						Content:          "final answer",
						ReasoningContent: []string{"solution found"},
					},
				},
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

	// Create client
	client := NewReasonerClient(ModelClientConfig{
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

	var contents []string
	var reasonings []string
	for resp := range respChan {
		contents = append(contents, resp.Choices[0].Message.Content)
		reasonings = append(reasonings, resp.Choices[0].Message.ReasoningContent...)
	}

	assert.Equal(t, []string{"thinking...", "final answer"}, contents)
	assert.Equal(t, []string{"analyzing problem", "solution found"}, reasonings)
}
