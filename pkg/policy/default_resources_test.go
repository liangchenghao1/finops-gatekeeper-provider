package policy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/zapr"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
	"go.uber.org/zap"
)

func TestDefaultResourcesMutator_Mutate(t *testing.T) {
	// 创建日志记录器
	zapLog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zapLog)

	// 创建DefaultResourcesMutator实例
	mutator := NewDefaultResourcesMutator(logger)

	// 创建测试请求数据
	providerRequest := externaldata.ProviderRequest{
		APIVersion: "externaldata.gatekeeper.sh/v1alpha1",
		Kind:       "ProviderRequest",
		Request: externaldata.Request{
			Keys: []string{"pod1", "pod2"},
		},
	}

	// 将请求数据转换为JSON
	requestBody, err := json.Marshal(providerRequest)
	if err != nil {
		t.Fatalf("Failed to marshal provider request: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", "/mutate", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	rr := httptest.NewRecorder()

	// 调用Mutate方法
	mutator.Mutate(rr, req)

	// 检查响应状态码
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// 解析响应
	var providerResponse externaldata.ProviderResponse
	err = json.Unmarshal(rr.Body.Bytes(), &providerResponse)
	if err != nil {
		t.Fatalf("Failed to unmarshal provider response: %v", err)
	}

	// 验证响应内容
	if providerResponse.APIVersion != "externaldata.gatekeeper.sh/v1alpha1" {
		t.Errorf("Expected APIVersion %s, got %s", "externaldata.gatekeeper.sh/v1alpha1", providerResponse.APIVersion)
	}
	
	if providerResponse.Kind != "ProviderResponse" {
		t.Errorf("Expected Kind %s, got %s", "ProviderResponse", providerResponse.Kind)
	}
	
	if providerResponse.Response.Idempotent != true {
		t.Errorf("Expected Idempotent %v, got %v", true, providerResponse.Response.Idempotent)
	}
	
	if len(providerResponse.Response.Items) != 2 {
		t.Errorf("Expected %d items, got %d", 2, len(providerResponse.Response.Items))
	}

	// 验证每个项目
	for i, item := range providerResponse.Response.Items {
		if item.Key != providerRequest.Request.Keys[i] {
			t.Errorf("Expected key %s, got %s", providerRequest.Request.Keys[i], item.Key)
		}
		
		// 解析返回的值
		var resources map[string]interface{}
		err = json.Unmarshal([]byte(item.Value.(string)), &resources)
		if err != nil {
			t.Fatalf("Failed to unmarshal resources: %v", err)
		}
		
		// 验证返回的资源结构
		_, hasResources := resources["resources"]
		if !hasResources {
			t.Error("Expected resources field in response")
		}
	}
}

func TestDefaultResourcesMutator_Mutate_WrongMethod(t *testing.T) {
	// 创建日志记录器
	zapLog, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zapLog)

	// 创建DefaultResourcesMutator实例
	mutator := NewDefaultResourcesMutator(logger)

	// 创建GET请求（错误的方法）
	req, err := http.NewRequest("GET", "/mutate", nil)
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}

	// 创建响应记录器
	rr := httptest.NewRecorder()

	// 调用Mutate方法
	mutator.Mutate(rr, req)

	// 检查响应状态码
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// 解析响应
	var providerResponse externaldata.ProviderResponse
	err = json.Unmarshal(rr.Body.Bytes(), &providerResponse)
	if err != nil {
		t.Fatalf("Failed to unmarshal provider response: %v", err)
	}

	// 验证响应包含系统错误
	if providerResponse.Response.SystemError == "" {
		t.Error("Expected SystemError to be set")
	}
	
	if providerResponse.Response.Items != nil {
		t.Error("Expected Items to be nil when SystemError is set")
	}
}