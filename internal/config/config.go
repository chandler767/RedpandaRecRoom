// internal/config/config.go
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Provider struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`     // openai_compat | gemini | anthropic
	Model    string `yaml:"model"`
	Endpoint string `yaml:"endpoint"` // empty for gemini/anthropic (hardcoded in clients)
	Enabled  bool   `yaml:"enabled"`
}

type Region struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Mode     string `yaml:"mode"`     // local | gcr
	Endpoint string `yaml:"endpoint"` // GCR endpoint URL, empty in local mode
}

type GitConfig struct {
	AutoPush bool   `yaml:"auto_push"`
	Remote   string `yaml:"remote"`
	Branch   string `yaml:"branch"`
}

type Config struct {
	Mode         string            `yaml:"mode"`
	SearchString string            `yaml:"search_string"`
	CallsPerLLM  int               `yaml:"calls_per_llm"`
	Providers    []Provider        `yaml:"providers"`
	Regions      []Region          `yaml:"regions"`
	Git          GitConfig         `yaml:"git"`
	APIKeys      map[string]string `yaml:"-"` // from local config only
}

type localConfig struct {
	APIKeys map[string]string `yaml:"api_keys"`
}

// Load reads the base config and merges API keys from localPath.
// localPath may not exist — that is not an error.
func Load(basePath, localPath string) (*Config, error) {
	data, err := os.ReadFile(basePath)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.CallsPerLLM == 0 {
		cfg.CallsPerLLM = 5
	}

	data, err = os.ReadFile(localPath)
	if err == nil {
		var lc localConfig
		if err := yaml.Unmarshal(data, &lc); err != nil {
			return nil, err
		}
		cfg.APIKeys = lc.APIKeys
	}
	if cfg.APIKeys == nil {
		cfg.APIKeys = map[string]string{}
	}
	return &cfg, nil
}
