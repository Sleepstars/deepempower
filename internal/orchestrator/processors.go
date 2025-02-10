package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/codeium/deepempower/internal/logger"
	"github.com/codeium/deepempower/internal/modelbridge"
	"github.com/codeium/deepempower/internal/models"
	"github.com/codeium/deepempower/internal/config" // 导入 config 包
)

// NormalPreprocessor implements the preprocessing stage using Normal model
type NormalPreprocessor struct {
	promptTemplate string
	bridge         *modelbridge.ModelBridge
	Logger         *logger.Logger
	config         *config.ModelConfig // 添加 config 字段
}

func newNormalPreprocessor(prompt string, bridge *modelbridge.ModelBridge) *NormalPreprocessor {
	return &NormalPreprocessor{
		promptTemplate: prompt,
		bridge:         bridge,
		Logger:         logger.GetLogger().WithComponent("normal_preprocessor"),
		config:         &config.ModelConfig{}, // 初始化 config 字段
	}
}

func (p *NormalPreprocessor) Name() string {
	return "normal_preprocessor"
}

func (p *NormalPreprocessor) Execute(ctx context.Context, data *Payload) error {
	// Parse prompt template
	tmpl, err := template.New("prompt").Parse(p.promptTemplate)
	if (err != nil) {
		p.Logger.WithError(err).Error("Failed to parse prompt template")
		return fmt.Errorf("parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]interface{}{
		"UserInput": data.OriginalRequest.Messages[len(data.OriginalRequest.Messages)-1].Content,
	}); err != nil {
		p.Logger.WithError(err).Error("Failed to execute prompt template")
		return fmt.Errorf("execute template: %w", err)
	}

	// Create model request with the same model as the original request
	req := &models.ChatCompletionRequest{
		Model: data.OriginalRequest.Model,
		Messages: []models.ChatCompletionMessage{
			{Role: "system", Content: buf.String()},
			{Role: "user", Content: data.OriginalRequest.Messages[len(data.OriginalRequest.Messages)-1].Content},
		},
	}

	// Call model through bridge
	resp, err := p.bridge.CallNormal(ctx, req)
	if err != nil {
		p.Logger.WithError(err).Error("Failed to call Normal model")
		return fmt.Errorf("model call: %w", err)
	}

	// Store structured input for next stage
	data.IntermContent = resp.Choices[0].Message.Content
	p.Logger.Debug("Preprocessing completed successfully")
	return nil
}

// ReasonerEngine implements the reasoning stage using Reasoner model
type ReasonerEngine struct {
	promptTemplate string
	bridge         *modelbridge.ModelBridge
	Logger         *logger.Logger
	config         *config.ModelConfig // 添加 config 字段
}

func newReasonerEngine(prompt string, bridge *modelbridge.ModelBridge) *ReasonerEngine {
	return &ReasonerEngine{
		promptTemplate: prompt,
		bridge:         bridge,
		Logger:         logger.GetLogger().WithComponent("reasoner_engine"),
		config:         &config.ModelConfig{}, // 初始化 config 字段
	}
}

func (p *ReasonerEngine) Name() string {
	return "reasoner_engine"
}

func (p *ReasonerEngine) Execute(ctx context.Context, data *Payload) error {
	// Parse prompt template
	tmpl, err := template.New("prompt").Parse(p.promptTemplate)
	if err != nil {
		p.Logger.WithError(err).Error("Failed to parse prompt template")
		return fmt.Errorf("parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]interface{}{
		"StructuredInput": data.IntermContent,
	}); err != nil {
		p.Logger.WithError(err).Error("Failed to execute prompt template")
		return fmt.Errorf("execute template: %w", err)
	}

	// Create model request with the same model as the original request
	req := &models.ChatCompletionRequest{
		Model: data.OriginalRequest.Model,
		Messages: []models.ChatCompletionMessage{
			{Role: "system", Content: buf.String()},
			{Role: "user", Content: data.IntermContent},
		},
		Stream: true,
	}

	// Call model with streaming through bridge
	respChan, err := p.bridge.CallReasonerStream(ctx, req)
	if err != nil {
		p.Logger.WithError(err).Error("Failed to start streaming from Reasoner model")
		return fmt.Errorf("model call: %w", err) // Removed "start stream:" prefix
	}

	// Process streaming response
	var lastContent string
	reasoningCount := 0
	for resp := range respChan {
		if len(resp.Choices) > 0 {
			// Collect reasoning chain
			if len(resp.Choices[0].Message.ReasoningContent) > 0 {
				data.ReasoningChain = append(data.ReasoningChain, resp.Choices[0].Message.ReasoningContent...)
				reasoningCount++
				p.Logger.Debug("Received reasoning step %d", reasoningCount)
			}
			// Update content
			lastContent = resp.Choices[0].Message.Content
		}
	}

	// Store final content
	data.IntermContent = lastContent
	p.Logger.Debug("Reasoning completed with %d steps", reasoningCount)
	return nil
}

// NormalPostprocessor implements the postprocessing stage using Normal model
type NormalPostprocessor struct {
	promptTemplate string
	bridge         *modelbridge.ModelBridge
	Logger         *logger.Logger
	config         *config.ModelConfig // 添加 config 字段
}

func newNormalPostprocessor(prompt string, bridge *modelbridge.ModelBridge) *NormalPostprocessor {
	return &NormalPostprocessor{
		promptTemplate: prompt,
		bridge:         bridge,
		Logger:         logger.GetLogger().WithComponent("normal_postprocessor"),
		config:         &config.ModelConfig{}, // 初始化 config 字段
	}
}

func (p *NormalPostprocessor) Name() string {
	return "normal_postprocessor"
}

func (p *NormalPostprocessor) Execute(ctx context.Context, data *Payload) error {
	// Parse prompt template
	tmpl, err := template.New("prompt").Parse(p.promptTemplate)
	if err != nil {
		p.Logger.WithError(err).Error("Failed to parse prompt template")
		return fmt.Errorf("parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]interface{}{
		"ReasoningChain":     data.ReasoningChain,
		"IntermediateResult": data.IntermContent,
	}); err != nil {
		p.Logger.WithError(err).Error("Failed to execute prompt template")
		return fmt.Errorf("execute template: %w", err)
	}

	// Create model request with the same model as the original request
	req := &models.ChatCompletionRequest{
		Model: data.OriginalRequest.Model,
		Messages: []models.ChatCompletionMessage{
			{Role: "system", Content: buf.String()},
			{Role: "user", Content: data.IntermContent},
		},
	}

	// Call model through bridge
	resp, err := p.bridge.CallNormal(ctx, req)
	if err != nil {
		p.Logger.WithError(err).Error("Failed to call Normal model")
		return fmt.Errorf("model call: %w", err)
	}

	// Store final content
	data.FinalContent = resp.Choices[0].Message.Content
	p.Logger.Debug("Postprocessing completed successfully")
	return nil
}
