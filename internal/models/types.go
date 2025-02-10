package models

// ChatCompletionRequest represents an incoming chat completion request
type ChatCompletionRequest struct {
	Model       string                  `json:"model"`
	Messages    []ChatCompletionMessage `json:"messages"`
	Stream      bool                    `json:"stream,omitempty"`
	RequestID   string                  `json:"request_id"`
	Temperature float32                 `json:"temperature,omitempty"`
	MaxTokens   int                     `json:"max_tokens,omitempty"`
}

// ChatCompletionMessage represents a message in the chat
type ChatCompletionMessage struct {
	Role             string   `json:"role"`
	Content          string   `json:"content"`
	ReasoningContent []string `json:"reasoning_content,omitempty"`
}

// ChatCompletionChoice represents a completion choice
type ChatCompletionChoice struct {
	Message      ChatCompletionMessage `json:"message"`
	FinishReason string               `json:"finish_reason"`
}

// ChatCompletionResponse represents the response from the chat completion API
type ChatCompletionResponse struct {
	ID      string                `json:"id"`
	Object  string                `json:"object"`
	Created int64                 `json:"created"`
	Model   string                `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
}
