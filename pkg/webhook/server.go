package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
)

// Server 是Webhook服务器的结构体
type Server struct {
	server       *http.Server
	logger       logr.Logger
	mutatePath   string
	validatePath string
	handler      *Handler
}

// Config 用于配置Webhook服务器
type Config struct {
	Port         int
	MutatePath   string
	ValidatePath string
	Timeout      time.Duration
	Logger       logr.Logger
}

// Handler 处理Webhook请求的结构体
type Handler struct {
	Logger  logr.Logger
	Timeout time.Duration
}

// NewServer 创建一个新的Webhook服务器
func NewServer(config Config) *Server {
	if config.MutatePath == "" {
		config.MutatePath = "/mutate"
	}
	if config.ValidatePath == "" {
		config.ValidatePath = "/validate"
	}
	if config.Timeout == 0 {
		config.Timeout = 3 * time.Second
	}

	handler := &Handler{
		Logger:  config.Logger,
		Timeout: config.Timeout,
	}

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: mux,
	}

	s := &Server{
		server:       server,
		logger:       config.Logger,
		mutatePath:   config.MutatePath,
		validatePath: config.ValidatePath,
		handler:      handler,
	}

	// 注册处理函数
	mux.HandleFunc(config.MutatePath, handler.processTimeout(handler.Mutate, config.Timeout))
	mux.HandleFunc(config.ValidatePath, handler.processTimeout(handler.Validate, config.Timeout))

	return s
}

// Start 启动Webhook服务器
func (s *Server) Start() error {
	s.logger.Info("starting webhook server", "port", s.server.Addr)
	return s.server.ListenAndServe()
}

// Stop 停止Webhook服务器
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("stopping webhook server")
	return s.server.Shutdown(ctx)
}

// Mutate 处理mutating webhook请求
func (h *Handler) Mutate(w http.ResponseWriter, req *http.Request) {
	// 只接受POST请求
	if req.Method != http.MethodPost {
		h.sendResponse(nil, "only POST is allowed", w)
		return
	}

	// 读取请求体
	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		h.sendResponse(nil, fmt.Sprintf("unable to read request body: %v", err), w)
		return
	}

	// 解析请求体
	var providerRequest externaldata.ProviderRequest
	if err := json.Unmarshal(requestBody, &providerRequest); err != nil {
		h.sendResponse(nil, fmt.Sprintf("unable to unmarshal request body: %v", err), w)
		return
	}

	// TODO: 实现具体的mutating逻辑
	// 这里应该根据请求中的Keys生成对应的Values
	results := make([]externaldata.Item, 0)
	for _, key := range providerRequest.Request.Keys {
		// 示例实现：直接返回相同的key作为value
		results = append(results, externaldata.Item{
			Key:   key,
			Value: key, // 这里应该替换成实际的处理逻辑
		})
	}

	h.sendResponse(&results, "", w)
}

// Validate 处理validating webhook请求
func (h *Handler) Validate(w http.ResponseWriter, req *http.Request) {
	// 只接受POST请求
	if req.Method != http.MethodPost {
		h.sendResponse(nil, "only POST is allowed", w)
		return
	}

	// 读取请求体
	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		h.sendResponse(nil, fmt.Sprintf("unable to read request body: %v", err), w)
		return
	}

	// 解析请求体
	var providerRequest externaldata.ProviderRequest
	if err := json.Unmarshal(requestBody, &providerRequest); err != nil {
		h.sendResponse(nil, fmt.Sprintf("unable to unmarshal request body: %v", err), w)
		return
	}

	// TODO: 实现具体的validating逻辑
	// 这里应该根据请求中的Keys进行验证，并返回验证结果
	results := make([]externaldata.Item, 0)
	for _, key := range providerRequest.Request.Keys {
		// 示例实现：所有key都验证通过
		results = append(results, externaldata.Item{
			Key:   key,
			Value: "valid", // 或者其他表示验证通过的值
		})
	}

	h.sendResponse(&results, "", w)
}

// sendResponse 发送响应给Gatekeeper
func (h *Handler) sendResponse(results *[]externaldata.Item, systemErr string, w http.ResponseWriter) {
	response := externaldata.ProviderResponse{
		APIVersion: "externaldata.gatekeeper.sh/v1alpha1",
		Kind:       "ProviderResponse",
		Response: externaldata.Response{
			Idempotent: true,
		},
	}

	if results != nil {
		response.Response.Items = *results
	} else {
		response.Response.SystemError = systemErr
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		panic(err)
	}
}

// processTimeout 为http处理函数添加超时机制
func (h *Handler) processTimeout(handler http.HandlerFunc, duration time.Duration) http.HandlerFunc {
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
			h.sendResponse(nil, "operation timed out", w)
		case <-processDone:
		}
	}
}