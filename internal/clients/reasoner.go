package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sleepstars/deepempower/internal/models"
)

// ReasonerClient implements ModelClient for the Reasoner (R1) model
type ReasonerClient struct {
	config ModelClientConfig
	client *http.Client
}

// NewReasonerClient creates a new Reasoner model client
func NewReasonerClient(config ModelClientConfig) *ReasonerClient {
	return &ReasonerClient{
		config: config,
		client: &http.Client{},
	}
}

// filterDisabledParams removes parameters that are not supported by the Reasoner model
func (c *ReasonerClient) filterDisabledParams(req *models.ChatCompletionRequest) {
	// Create a copy of the request for modification
	reqMap := make(map[string]interface{})
	data, _ := json.Marshal(req)
	json.Unmarshal(data, &reqMap)

	// Remove disabled parameters
	for _, param := range c.config.DisabledParams {
		delete(reqMap, param)
	}

	// Update the request
	data, _ = json.Marshal(reqMap)
	json.Unmarshal(data, req)
}

func (c *ReasonerClient) Complete(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	// Remove unsupported parameters
	c.filterDisabledParams(req)

	// Set model from config if not specified
	if req.Model == "" {
		req.Model = c.config.Model
	}

	// Prepare request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/chat/completions", c.config.APIBase)
	// Ensure URL has scheme
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var result struct {
		Choices []struct {
			Message struct {
				Content          string   `json:"content"`
				ReasoningContent []string `json:"reasoning_content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to standard response format
	return &models.ChatCompletionResponse{
		Choices: []models.ChatCompletionChoice{
			{
				Message: models.ChatCompletionMessage{
					Role:             "assistant",
					Content:          result.Choices[0].Message.Content,
					ReasoningContent: result.Choices[0].Message.ReasoningContent,
				},
				FinishReason: "stop",
			},
		},
	}, nil
}

func (c *ReasonerClient) CompleteStream(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
	// Remove unsupported parameters
	c.filterDisabledParams(req)

	// Set model from config if not specified
	if req.Model == "" {
		req.Model = c.config.Model
	}

	resultChan := make(chan *models.ChatCompletionResponse)

	// Create request with streaming flag
	req.Stream = true

	// Prepare request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/chat/completions", c.config.APIBase)
	// Ensure URL has scheme
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	// Start goroutine to read streaming response
	go func() {
		defer close(resultChan)
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var chunk struct {
					Choices []struct {
						Message struct {
							Content          string   `json:"content"`
							ReasoningContent []string `json:"reasoning_content"`
						} `json:"message"`
					} `json:"choices"`
				}
				if err := decoder.Decode(&chunk); err != nil {
					// End of stream or error
					return
				}

				// Convert to standard response format
				resultChan <- &models.ChatCompletionResponse{
					Choices: []models.ChatCompletionChoice{
						{
							Message: models.ChatCompletionMessage{
								Role:             "assistant",
								Content:          chunk.Choices[0].Message.Content,
								ReasoningContent: chunk.Choices[0].Message.ReasoningContent,
							},
							FinishReason: "stop",
						},
					},
				}
			}
		}
	}()

	return resultChan, nil
}
