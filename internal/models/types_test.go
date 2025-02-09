package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChatCompletionRequestSerialization(t *testing.T) {
	req := &ChatCompletionRequest{
		Model: "test-model",
		Messages: []ChatCompletionMessage{
			{
				Role:    "user",
				Content: "test message",
				ReasoningContent: []string{
					"step 1",
					"step 2",
				},
			},
		},
		Stream:    true,
		RequestID: "test-123",
	}

	// Test marshaling
	data, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"model":"test-model"`)
	assert.Contains(t, string(data), `"role":"user"`)
	assert.Contains(t, string(data), `"content":"test message"`)
	assert.Contains(t, string(data), `"reasoning_content":["step 1","step 2"]`)
	assert.Contains(t, string(data), `"stream":true`)
	assert.Contains(t, string(data), `"request_id":"test-123"`)

	// Test unmarshaling
	var newReq ChatCompletionRequest
	err = json.Unmarshal(data, &newReq)
	assert.NoError(t, err)
	assert.Equal(t, req.Model, newReq.Model)
	assert.Equal(t, req.Messages[0].Role, newReq.Messages[0].Role)
	assert.Equal(t, req.Messages[0].Content, newReq.Messages[0].Content)
	assert.Equal(t, req.Messages[0].ReasoningContent, newReq.Messages[0].ReasoningContent)
	assert.Equal(t, req.Stream, newReq.Stream)
	assert.Equal(t, req.RequestID, newReq.RequestID)
}

func TestChatCompletionResponseSerialization(t *testing.T) {
	resp := &ChatCompletionResponse{
		Choices: []ChatCompletionChoice{
			{
				Message: ChatCompletionMessage{
					Role:    "assistant",
					Content: "test response",
					ReasoningContent: []string{
						"reasoning step 1",
						"reasoning step 2",
					},
				},
				FinishReason: "stop",
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(resp)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"role":"assistant"`)
	assert.Contains(t, string(data), `"content":"test response"`)
	assert.Contains(t, string(data), `"reasoning_content":["reasoning step 1","reasoning step 2"]`)
	assert.Contains(t, string(data), `"finish_reason":"stop"`)

	// Test unmarshaling
	var newResp ChatCompletionResponse
	err = json.Unmarshal(data, &newResp)
	assert.NoError(t, err)
	assert.Equal(t, resp.Choices[0].Message.Role, newResp.Choices[0].Message.Role)
	assert.Equal(t, resp.Choices[0].Message.Content, newResp.Choices[0].Message.Content)
	assert.Equal(t, resp.Choices[0].Message.ReasoningContent, newResp.Choices[0].Message.ReasoningContent)
	assert.Equal(t, resp.Choices[0].FinishReason, newResp.Choices[0].FinishReason)
}

func TestChatCompletionMessageValidation(t *testing.T) {
	testCases := []struct {
		name    string
		message ChatCompletionMessage
		isValid bool
	}{
		{
			name: "Valid user message",
			message: ChatCompletionMessage{
				Role:    "user",
				Content: "test message",
			},
			isValid: true,
		},
		{
			name: "Valid assistant message with reasoning",
			message: ChatCompletionMessage{
				Role:    "assistant",
				Content: "test response",
				ReasoningContent: []string{
					"step 1",
					"step 2",
				},
			},
			isValid: true,
		},
		{
			name: "Valid system message",
			message: ChatCompletionMessage{
				Role:    "system",
				Content: "system instruction",
			},
			isValid: true,
		},
		{
			name: "Empty role",
			message: ChatCompletionMessage{
				Content: "test message",
			},
			isValid: false,
		},
		{
			name: "Empty content",
			message: ChatCompletionMessage{
				Role: "user",
			},
			isValid: false,
		},
		{
			name: "Invalid role",
			message: ChatCompletionMessage{
				Role:    "invalid",
				Content: "test message",
			},
			isValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.message)
			assert.NoError(t, err)

			var msg ChatCompletionMessage
			err = json.Unmarshal(data, &msg)
			if tc.isValid {
				assert.NoError(t, err)
				assert.Equal(t, tc.message.Role, msg.Role)
				assert.Equal(t, tc.message.Content, msg.Content)
				assert.Equal(t, tc.message.ReasoningContent, msg.ReasoningContent)
			}
		})
	}
}
