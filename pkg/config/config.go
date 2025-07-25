package config

// WebhookConfig 定义单个webhook的配置
type WebhookConfig struct {
	// Name webhook名称
	Name string `json:"name" yaml:"name"`
	
	// Path webhook路径
	Path string `json:"path" yaml:"path"`
	
	// Type webhook类型，可以是"mutating"或"validating"
	Type string `json:"type" yaml:"type"`
	
	// Enabled 是否启用该webhook
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// ServerConfig 定义服务器配置
type ServerConfig struct {
	// Port 服务器端口
	Port int `json:"port" yaml:"port"`
	
	// Timeout 超时时间(秒)
	Timeout int `json:"timeout" yaml:"timeout"`
}

// Config 定义完整的配置结构
type Config struct {
	// Server 服务器配置
	Server ServerConfig `json:"server" yaml:"server"`
	
	// Webhooks webhook配置列表
	Webhooks []WebhookConfig `json:"webhooks" yaml:"webhooks"`
}