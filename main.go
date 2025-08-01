package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/config"
	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/policy"
	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/server"
)

var (
	log      logr.Logger
	certFile = "cert/server.crt"
	keyFile  = "cert/server.key"
)

func main() {
	// 初始化日志
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("unable to initialize logger: %v", err))
	}
	log = zapr.NewLogger(zapLog)
	log = log.WithName("finops-gatekeeper-provider")

	// 加载配置文件
	cfg, err := loadConfig()
	if err != nil {
		log.Error(err, "failed to load config")
		os.Exit(1)
	}

	// 创建webhook服务器配置
	serverConfig := server.Config{
		Port:    cfg.Server.Port,
		Timeout: time.Duration(cfg.Server.Timeout) * time.Second,
		Logger:  log,
	}

	// 创建webhook服务器
	s := server.NewServer(serverConfig)

	// 根据配置文件注册webhook
	if err := policy.RegisterWebhooksFromConfig(s, cfg, log); err != nil {
		log.Error(err, "failed to register webhooks from config")
		os.Exit(1)
	}

	// 打印已注册的webhook信息
	log.Info("registered mutating webhooks", "paths", s.ListMutatingWebhooks())
	log.Info("registered validating webhooks", "paths", s.ListValidatingWebhooks())

	// 启动服务器
	go func() {
		log.Info("starting server", "port", cfg.Server.Port)
		// 检查是否使用TLS
		useTLS := "true"
		if value := os.Getenv("USE_TLS"); value != "" {
			useTLS = value
		}

		if useTLS == "true" {
			log.Info("starting HTTPS server with TLS", "certFile", certFile, "keyFile", keyFile)
			// 检查证书文件是否存在
			if _, err := os.Stat(certFile); os.IsNotExist(err) {
				log.Error(err, "TLS certificate file not found", "certFile", certFile)
				panic(fmt.Sprintf("TLS certificate file not found: %s", certFile))
			}
			if _, err := os.Stat(keyFile); os.IsNotExist(err) {
				log.Error(err, "TLS key file not found", "keyFile", keyFile)
				panic(fmt.Sprintf("TLS key file not found: %s", keyFile))
			}

			if err := s.StartTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				log.Error(err, "HTTPS server failed to start")
				panic(err)
			}
		} else {
			log.Info("starting HTTP server")
			if err := s.Start(); err != nil && err != http.ErrServerClosed {
				log.Error(err, "server failed to start")
				panic(err)
			}
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down server...")

	// 优雅地关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Stop(ctx); err != nil {
		log.Error(err, "server forced to shutdown")
	}

	log.Info("server exited")
}

// loadConfig 加载配置文件
func loadConfig() (*config.Config, error) {
	// 默认配置
	defaultConfig := &config.Config{
		Server: config.ServerConfig{
			Port:    8090,
			Timeout: 5,
		},
		Webhooks: []config.WebhookConfig{
			{
				Name:    "default-resources-mutator",
				Path:    "/mutate/resources",
				Type:    "mutating",
				Enabled: true,
			},
			{
				Name:    "default-labels-mutator",
				Path:    "/mutate/labels",
				Type:    "mutating",
				Enabled: true,
			},
			{
				Name:    "resources-validator",
				Path:    "/validate/resources",
				Type:    "validating",
				Enabled: true,
			},
			{
				Name:    "labels-validator",
				Path:    "/validate/labels",
				Type:    "validating",
				Enabled: true,
			},
		},
	}

	// 检查是否提供了配置文件路径
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml" // 默认配置文件路径
	}

	// 如果配置文件存在，则加载它
	if _, err := os.Stat(configPath); err == nil {
		log.Info("loading config from file", "path", configPath)
		cfg, err := config.Load(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
		}
		return cfg, nil
	}

	// 否则使用默认配置
	log.Info("config file not found, using default config", "path", configPath)
	return defaultConfig, nil
}
