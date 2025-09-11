package config

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// LoadFromYAML 从YAML文件加载配置
func LoadFromYAML(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &config, nil
}

// LoadFromJSON 从JSON文件加载配置
func LoadFromJSON(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &config, nil
}

// Load 从文件加载配置，根据文件扩展名自动选择解析方式
func Load(filename string) (*Config, error) {
	switch {
	case len(filename) >= 5 && filename[len(filename)-5:] == ".yaml":
		return LoadFromYAML(filename)
	case len(filename) >= 4 && filename[len(filename)-4:] == ".yml":
		return LoadFromYAML(filename)
	case len(filename) >= 5 && filename[len(filename)-5:] == ".json":
		return LoadFromJSON(filename)
	default:
		return nil, fmt.Errorf("unsupported config file format")
	}
}