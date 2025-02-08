package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/codeium/deepempower/internal/logger"
	"github.com/codeium/deepempower/internal/modelbridge"
	"github.com/codeium/deepempower/internal/models"
)

// NormalPreprocessor implements the preprocessing stage using Normal model
type NormalPreprocessor struct {
	promptTemplate string
	bridge         *modelbridge.ModelBridge
	logger         *logger.Logger
}

func newNormalPreprocessor(prompt string, bridge *modelbridge.ModelBridge) *NormalPreprocessor {
	return &NormalPreprocessor{
		promptTemplate: prompt,
		bridge:         bridge,
		logger:         logger.GetLogger().WithComponent("normal_preprocessor"),
	}
}

func (p *NormalPreprocessor) Name() string {
	return "normal_preprocessor"
}

func (p *NormalPreprocessor) Execute(ctx context.Context, data *Payload) error {
	p.logger.Debug("Starting preprocessing stage")

	// Parse prompt template
	tmpl, err := template.New("prompt").Parse(p.promptTemplate)
	if err != nil {
		p.logger.WithError(err).Error("Failed to parse prompt template")
		return fmt.Errorf("parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]interface{}{
		"UserInput": data.OriginalRequest.Messages[len(data.OriginalRequest.Messages)-1].Content,
	}); err != nil {
		p.logger.WithError(err).Error("Failed to execute prompt template")
		return fmt.Errorf("execute template: %w", err)
	}

	p.logger.Debug("Prepared prompt for Normal model")

	// Create model request
	req := &models.ChatCompletionRequest{
		Messages: []models.ChatCompletionMessage{
			{Role: "system", Content: buf.String()},
			{Role: "user", Content: data.OriginalRequest.Messages[len(data.OriginalRequest.Messages)-1].Content},
		},
	}

	// Call model through bridge
	p.logger.Debug("Calling Normal model")
	resp, err := p.bridge.CallNormal(ctx, req)
	if err != nil {
		p.logger.WithError(err).Error("Failed to call Normal model")
		return fmt.Errorf("model call: %w", err)
	}

	// Store structured input for next stage
	data.IntermContent = resp.Choices[0].Message.Content
	p.logger.Debug("Preprocessing completed successfully")
	return nil
}

// ReasonerEngine implements the reasoning stage using Reasoner model
type ReasonerEngine struct {
	promptTemplate string
	bridge         *modelbridge.ModelBridge
	logger         *logger.Logger
}

func newReasonerEngine(prompt string, bridge *modelbridge.ModelBridge) *ReasonerEngine {
	return &ReasonerEngine{
		promptTemplate: prompt,
		bridge:         bridge,
		logger:         logger.GetLogger().WithComponent("reasoner_engine"),
	}
}

func (p *ReasonerEngine) Name() string {
	return "reasoner_engine"
}

func (p *ReasonerEngine) Execute(ctx context.Context, data *Payload) error {
	p.logger.Debug("Starting reasoning stage")

	// Parse prompt template
	tmpl, err := template.New("prompt").Parse(p.promptTemplate)
	if err != nil {
		p.logger.WithError(err).Error("Failed to parse prompt template")
		return fmt.Errorf("parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]interface{}{
		"StructuredInput": data.IntermContent,
	}); err != nil {
		p.logger.WithError(err).Error("Failed to execute prompt template")
		return fmt.Errorf("execute template: %w", err)
	}

	p.logger.Debug("Prepared prompt for Reasoner model")

	// Create model request
	req := &models.ChatCompletionRequest{
		Messages: []models.ChatCompletionMessage{
			{Role: "system", Content: buf.String()},
			{Role: "user", Content: data.IntermContent},
		},
	}

	// Call model with streaming through bridge
	p.logger.Debug("Starting streaming call to Reasoner model")
	respChan, err := p.bridge.CallReasonerStream(ctx, req)
	if err != nil {
		p.logger.WithError(err).Error("Failed to start streaming from Reasoner model")
		return fmt.Errorf("model call: %w", err)
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
				p.logger.Debug("Received reasoning step %d", reasoningCount)
			}
			// Update content
			lastContent = resp.Choices[0].Message.Content
		}
	}

	// Store final content
	data.IntermContent = lastContent
	p.logger.Debug("Reasoning completed with %d steps", reasoningCount)
	return nil
}

// NormalPostprocessor implements the postprocessing stage using Normal model
type NormalPostprocessor struct {
	promptTemplate string
	bridge         *modelbridge.ModelBridge
	logger         *logger.Logger
}

func newNormalPostprocessor(prompt string, bridge *modelbridge.ModelBridge) *NormalPostprocessor {
	return &NormalPostprocessor{
		promptTemplate: prompt,
		bridge:         bridge,
		logger:         logger.GetLogger().WithComponent("normal_postprocessor"),
	}
}

func (p *NormalPostprocessor) Name() string {
	return "normal_postprocessor"
}

func (p *NormalPostprocessor) Execute(ctx context.Context, data *Payload) error {
	p.logger.Debug("Starting postprocessing stage")

	// Parse prompt template
	tmpl, err := template.New("prompt").Parse(p.promptTemplate)
	if err != nil {
		p.logger.WithError(err).Error("Failed to parse prompt template")
		return fmt.Errorf("parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]interface{}{
		"ReasoningChain":     data.ReasoningChain,
		"IntermediateResult": data.IntermContent,
	}); err != nil {
		p.logger.WithError(err).Error("Failed to execute prompt template")
		return fmt.Errorf("execute template: %w", err)
	}

	p.logger.Debug("Prepared prompt for Normal model with %d reasoning steps", len(data.ReasoningChain))

	// Create model request
	req := &models.ChatCompletionRequest{
		Messages: []models.ChatCompletionMessage{
			{Role: "system", Content: buf.String()},
			{Role: "user", Content: data.IntermContent},
		},
	}

	// Call model through bridge
	p.logger.Debug("Calling Normal model for final response")
	resp, err := p.bridge.CallNormal(ctx, req)
	if err != nil {
		p.logger.WithError(err).Error("Failed to call Normal model")
		return fmt.Errorf("model call: %w", err)
	}

	// Store final content
	data.FinalContent = resp.Choices[0].Message.Content
	p.logger.Debug("Postprocessing completed successfully")
	return nil
}
