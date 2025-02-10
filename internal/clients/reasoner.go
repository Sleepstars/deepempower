package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sleepstars/deepempower/internal/models"
)

// ReasonerClient implements ModelClient for the Reasoner (R1) model
type ReasonerClient struct {
	config ModelClientConfig
	client *openai.Client
}

// NewReasonerClient creates a new Reasoner model client
func NewReasonerClient(config ModelClientConfig) *ReasonerClient {
	clientConfig := openai.DefaultConfig("")
	clientConfig.BaseURL = config.APIBase
	if !strings.HasPrefix(clientConfig.BaseURL, "http://") && !strings.HasPrefix(clientConfig.BaseURL, "https://") {
		clientConfig.BaseURL = "http://" + clientConfig.BaseURL
	}

	return &ReasonerClient{
		config: config,
		client: openai.NewClientWithConfig(clientConfig),
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

	// Convert to openai request format
	openaiReq := openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: make([]openai.ChatCompletionMessage, len(req.Messages)),
	}

	// Convert messages
	for i, msg := range req.Messages {
		openaiReq.Messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Call OpenAI API
	resp, err := c.client.CreateChatCompletion(ctx, openaiReq)
	if err != nil {
		return nil, fmt.Errorf("create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	// Convert response back to our format
	return &models.ChatCompletionResponse{
		Choices: []models.ChatCompletionChoice{
			{
				Message: models.ChatCompletionMessage{
					Role:    resp.Choices[0].Message.Role,
					Content: resp.Choices[0].Message.Content,
				},
				FinishReason: string(resp.Choices[0].FinishReason),
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

	// Convert to openai request format
	openaiReq := openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: make([]openai.ChatCompletionMessage, len(req.Messages)),
		Stream:   true,
	}

	// Convert messages
	for i, msg := range req.Messages {
		openaiReq.Messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Create stream
	stream, err := c.client.CreateChatCompletionStream(ctx, openaiReq)
	if err != nil {
		return nil, fmt.Errorf("create chat completion stream: %w", err)
	}

	resultChan := make(chan *models.ChatCompletionResponse)

	// Start goroutine to read streaming response
	go func() {
		defer close(resultChan)
		defer stream.Close()

		var contentBuilder strings.Builder
		var partialContent []string

		for {
			select {
			case <-ctx.Done():
				return
			default:
				resp, err := stream.Recv()
				if err != nil {
					// End of stream or error
					return
				}

				if len(resp.Choices) > 0 && resp.Choices[0].Delta.Content != "" {
					content := resp.Choices[0].Delta.Content
					contentBuilder.WriteString(content)
					partialContent = append(partialContent, content)

					// Convert to standard response format
					resultChan <- &models.ChatCompletionResponse{
						Choices: []models.ChatCompletionChoice{
							{
								Message: models.ChatCompletionMessage{
									Role:             resp.Choices[0].Delta.Role,
									Content:          content,
									ReasoningContent: []string{},
								},
								FinishReason: string(resp.Choices[0].FinishReason),
							},
						},
					}
				}
			}
		}
	}()

	return resultChan, nil
}
