package config

// PipelineConfig represents the configuration for a processing pipeline
type PipelineConfig struct {
	// Prompts contains the prompt templates for each stage
	Prompts PromptConfig `yaml:"prompts"`
	// Models contains the model configurations
	Models ModelConfig `yaml:"models"`
}

// PromptConfig contains prompt templates for different stages
type PromptConfig struct {
	PreProcess  string `yaml:"pre_process"`
	Reasoning   string `yaml:"reasoning"`
	PostProcess string `yaml:"post_process"`
}

// ModelConfig contains configurations for different models
type ModelConfig struct {
	Normal struct {
		APIBase       string                 `yaml:"api_base"`
		DefaultParams map[string]interface{} `yaml:"default_params"`
	} `yaml:"normal"`
	Reasoner struct {
		APIBase         string   `yaml:"api_base"`
		DisabledParams []string `yaml:"disabled_params"`
	} `yaml:"reasoner"`
}
