package utils

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
)

func TestReadProviderRequest_Success(t *testing.T) {
	// 创建测试请求数据
	providerRequest := externaldata.ProviderRequest{
		APIVersion: "externaldata.gatekeeper.sh/v1alpha1",
		Kind:       "ProviderRequest",
		Request: externaldata.Request{
			Keys: []string{"key1", "key2"},
		},
	}

	// 将请求数据转换为JSON
	requestBody, err := json.Marshal(providerRequest)
	if err != nil {
		t.Fatalf("Failed to marshal provider request: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", "/test", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	rr := httptest.NewRecorder()

	// 调用ReadProviderRequest函数
	result, err := ReadProviderRequest(rr, req)
	if err != nil {
		t.Fatalf("ReadProviderRequest failed: %v", err)
	}

	// 验证结果
	if result == nil {
		t.Fatal("Expected provider request, got nil")
	}

	if result.APIVersion != providerRequest.APIVersion {
		t.Errorf("Expected APIVersion %s, got %s", providerRequest.APIVersion, result.APIVersion)
	}

	if result.Kind != providerRequest.Kind {
		t.Errorf("Expected Kind %s, got %s", providerRequest.Kind, result.Kind)
	}

	if len(result.Request.Keys) != len(providerRequest.Request.Keys) {
		t.Errorf("Expected %d keys, got %d", len(providerRequest.Request.Keys), len(result.Request.Keys))
	}

	for i, key := range result.Request.Keys {
		if key != providerRequest.Request.Keys[i] {
			t.Errorf("Expected key %s, got %s", providerRequest.Request.Keys[i], key)
		}
	}
}

func TestReadProviderRequest_WrongMethod(t *testing.T) {
	// 创建HTTP GET请求（错误的方法）
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}

	// 创建响应记录器
	rr := httptest.NewRecorder()

	// 调用ReadProviderRequest函数
	result, err := ReadProviderRequest(rr, req)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result, got non-nil")
	}

	// 验证响应
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

func TestReadProviderRequest_InvalidJSON(t *testing.T) {
	// 创建无效的JSON数据
	invalidJSON := []byte("{ invalid json }")

	// 创建HTTP请求
	req, err := http.NewRequest("POST", "/test", bytes.NewBuffer(invalidJSON))
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	rr := httptest.NewRecorder()

	// 调用ReadProviderRequest函数
	result, err := ReadProviderRequest(rr, req)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Error("Expected nil result, got non-nil")
	}

	// 验证响应
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

func TestSendResponse_Items(t *testing.T) {
	// 创建测试数据
	items := []externaldata.Item{
		{
			Key:   "key1",
			Value: "value1",
		},
		{
			Key:   "key2",
			Value: "value2",
		},
	}

	// 创建响应记录器
	rr := httptest.NewRecorder()

	// 调用SendResponse函数
	SendResponse(rr, &items, "")

	// 验证响应
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// 解析响应
	var providerResponse externaldata.ProviderResponse
	err := json.Unmarshal(rr.Body.Bytes(), &providerResponse)
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

	if len(providerResponse.Response.Items) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(providerResponse.Response.Items))
	}

	for i, item := range providerResponse.Response.Items {
		if item.Key != items[i].Key {
			t.Errorf("Expected key %s, got %s", items[i].Key, item.Key)
		}

		if item.Value != items[i].Value {
			t.Errorf("Expected value %s, got %s", items[i].Value, item.Value)
		}
	}

	if providerResponse.Response.SystemError != "" {
		t.Error("Expected SystemError to be empty")
	}
}

func TestSendResponse_SystemError(t *testing.T) {
	// 创建系统错误消息
	systemErr := "test system error"

	// 创建响应记录器
	rr := httptest.NewRecorder()

	// 调用SendResponse函数
	SendResponse(rr, nil, systemErr)

	// 验证响应
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	// 解析响应
	var providerResponse externaldata.ProviderResponse
	err := json.Unmarshal(rr.Body.Bytes(), &providerResponse)
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

	if providerResponse.Response.Items != nil {
		t.Error("Expected Items to be nil")
	}

	if providerResponse.Response.SystemError != systemErr {
		t.Errorf("Expected SystemError %s, got %s", systemErr, providerResponse.Response.SystemError)
	}
}
