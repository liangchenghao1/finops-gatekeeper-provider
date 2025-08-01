package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// Server 是Webhook服务器的结构体
type Server struct {
	server *http.Server
	logger logr.Logger
	mux    *http.ServeMux

	// 用于保护webhook映射的互斥锁
	mu sync.RWMutex

	// 存储已注册的webhook
	mutatingWebhooks   map[string]MutatingWebhook
	validatingWebhooks map[string]ValidatingWebhook

	// 默认超时时间
	timeout time.Duration
}

// Config 用于配置Webhook服务器
type Config struct {
	Port    int
	Timeout time.Duration
	Logger  logr.Logger
}

// NewServer 创建一个新的Webhook服务器
func NewServer(config Config) *Server {
	if config.Timeout == 0 {
		config.Timeout = 3 * time.Second
	}

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: mux,
	}

	s := &Server{
		server:             server,
		logger:             config.Logger,
		mux:                mux,
		mutatingWebhooks:   make(map[string]MutatingWebhook),
		validatingWebhooks: make(map[string]ValidatingWebhook),
		timeout:            config.Timeout,
	}

	// 设置默认的处理函数
	mux.HandleFunc("/", s.defaultHandler)

	return s
}

// RegisterMutatingWebhook 注册Mutating Webhook
func (s *Server) RegisterMutatingWebhook(path string, webhook MutatingWebhook) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查路径是否已被占用
	if _, exists := s.mutatingWebhooks[path]; exists {
		return fmt.Errorf("mutating webhook already registered for path: %s", path)
	}

	// 注册webhook
	s.mutatingWebhooks[path] = webhook

	// 注册HTTP处理函数
	s.mux.HandleFunc(path, s.processTimeout(webhook.Mutate, s.timeout))

	s.logger.Info("registered mutating webhook", "path", path)
	return nil
}

// RegisterValidatingWebhook 注册Validating Webhook
func (s *Server) RegisterValidatingWebhook(path string, webhook ValidatingWebhook) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查路径是否已被占用
	if _, exists := s.validatingWebhooks[path]; exists {
		return fmt.Errorf("validating webhook already registered for path: %s", path)
	}

	// 注册webhook
	s.validatingWebhooks[path] = webhook

	// 注册HTTP处理函数
	s.mux.HandleFunc(path, s.processTimeout(webhook.Validate, s.timeout))

	s.logger.Info("registered validating webhook", "path", path)
	return nil
}

// UnregisterMutatingWebhook 取消注册Mutating Webhook
func (s *Server) UnregisterMutatingWebhook(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查路径是否存在
	if _, exists := s.mutatingWebhooks[path]; !exists {
		return fmt.Errorf("no mutating webhook registered for path: %s", path)
	}

	// 取消注册webhook
	delete(s.mutatingWebhooks, path)

	s.logger.Info("unregistered mutating webhook", "path", path)
	return nil
}

// UnregisterValidatingWebhook 取消注册Validating Webhook
func (s *Server) UnregisterValidatingWebhook(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查路径是否存在
	if _, exists := s.validatingWebhooks[path]; !exists {
		return fmt.Errorf("no validating webhook registered for path: %s", path)
	}

	// 取消注册webhook
	delete(s.validatingWebhooks, path)

	s.logger.Info("unregistered validating webhook", "path", path)
	return nil
}

// ListMutatingWebhooks 列出所有已注册的Mutating Webhook路径
func (s *Server) ListMutatingWebhooks() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	paths := make([]string, 0, len(s.mutatingWebhooks))
	for path := range s.mutatingWebhooks {
		paths = append(paths, path)
	}

	return paths
}

// ListValidatingWebhooks 列出所有已注册的Validating Webhook路径
func (s *Server) ListValidatingWebhooks() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	paths := make([]string, 0, len(s.validatingWebhooks))
	for path := range s.validatingWebhooks {
		paths = append(paths, path)
	}

	return paths
}

// Start 启动Webhook服务器
func (s *Server) Start() error {
	s.logger.Info("starting webhook server", "port", s.server.Addr)
	return s.server.ListenAndServe()
}

// StartTLS 启动HTTPS Webhook服务器
func (s *Server) StartTLS(certFile, keyFile string) error {
	s.logger.Info("starting webhook server with TLS", "port", s.server.Addr)
	return s.server.ListenAndServeTLS(certFile, keyFile)
}

// Stop 停止Webhook服务器
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("stopping webhook server")
	return s.server.Shutdown(ctx)
}

// defaultHandler 默认处理函数
func (s *Server) defaultHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("404 - Path not found"))
}

// processTimeout 为http处理函数添加超时机制
func (s *Server) processTimeout(handler http.HandlerFunc, duration time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), duration)
		defer cancel()

		r = r.WithContext(ctx)

		processDone := make(chan bool)
		go func() {
			handler(w, r)
			processDone <- true
		}()

		select {
		case <-ctx.Done():
			// 发送超时响应
			w.WriteHeader(http.StatusOK)
			response := `{"apiVersion":"externaldata.gatekeeper.sh/v1alpha1","kind":"ProviderResponse","response":{"systemError":"operation timed out","idempotent":true}}`
			_, _ = w.Write([]byte(response))
		case <-processDone:
		}
	}
}
