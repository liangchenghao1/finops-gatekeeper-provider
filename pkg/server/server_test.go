package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

// mockMutator 是一个模拟的MutatingWebhook
type mockMutator struct {
	called bool
}

func (m *mockMutator) Mutate(w http.ResponseWriter, r *http.Request) {
	m.called = true
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("mutator called"))
}

// mockValidator 是一个模拟的ValidatingWebhook
type mockValidator struct {
	called bool
}

func (m *mockValidator) Validate(w http.ResponseWriter, r *http.Request) {
	m.called = true
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("validator called"))
}

func TestServer_StartAndStop(t *testing.T) {
	// 创建日志记录器
	zapLog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zapLog)

	// 创建服务器配置
	config := Config{
		Port:    8091, // 使用不同的端口避免冲突
		Timeout: 5 * time.Second,
		Logger:  logger,
	}

	// 创建服务器
	s := NewServer(config)

	// 注册模拟的webhook
	mutator := &mockMutator{}
	validator := &mockValidator{}
	s.RegisterMutatingWebhook("/mutate", mutator)
	s.RegisterValidatingWebhook("/validate", validator)

	// 在goroutine中启动服务器
	go func() {
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server failed to start: %v", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 测试mutate端点
	mutateReq, err := http.NewRequest("POST", "http://localhost:8091/mutate", nil)
	if err != nil {
		t.Fatalf("Failed to create mutate request: %v", err)
	}

	mutateResp := httptest.NewRecorder()
	// 使用默认的服务器处理程序而不是mux
	s.server.Handler.ServeHTTP(mutateResp, mutateReq)

	if mutateResp.Code != http.StatusOK {
		t.Errorf("Expected mutate response code %d, got %d", http.StatusOK, mutateResp.Code)
	}

	if !mutator.called {
		t.Error("Mutator was not called")
	}

	// 测试validate端点
	validateReq, err := http.NewRequest("POST", "http://localhost:8091/validate", nil)
	if err != nil {
		t.Fatalf("Failed to create validate request: %v", err)
	}

	validateResp := httptest.NewRecorder()
	// 使用默认的服务器处理程序而不是mux
	s.server.Handler.ServeHTTP(validateResp, validateReq)

	if validateResp.Code != http.StatusOK {
		t.Errorf("Expected validate response code %d, got %d", http.StatusOK, validateResp.Code)
	}

	if !validator.called {
		t.Error("Validator was not called")
	}

	// 停止服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Stop(ctx); err != nil {
		t.Errorf("Server failed to stop: %v", err)
	}
}

func TestServer_WrongMethod(t *testing.T) {
	// 创建日志记录器
	zapLog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zapLog)

	// 创建服务器配置
	config := Config{
		Port:    8092, // 使用不同的端口避免冲突
		Timeout: 5 * time.Second,
		Logger:  logger,
	}

	// 创建服务器
	s := NewServer(config)

	// 注册模拟的webhook
	mutator := &mockMutator{}
	s.RegisterMutatingWebhook("/mutate", mutator)

	// 在goroutine中启动服务器
	go func() {
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server failed to start: %v", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 测试不存在的路径（应该返回404）
	getReq, err := http.NewRequest("GET", "http://localhost:8092/nonexistent", nil)
	if err != nil {
		t.Fatalf("Failed to create GET request: %v", err)
	}

	getResp := httptest.NewRecorder()
	// 使用默认的服务器处理程序而不是mux
	s.server.Handler.ServeHTTP(getResp, getReq)

	if getResp.Code != http.StatusNotFound {
		t.Errorf("Expected response code %d, got %d", http.StatusNotFound, getResp.Code)
	}

	// 停止服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Stop(ctx); err != nil {
		t.Errorf("Server failed to stop: %v", err)
	}
}