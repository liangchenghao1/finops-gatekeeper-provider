package policy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/utils"
	"github.com/go-logr/logr"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
)

func TestAlibabaCloudBillingValidator_ParseKey(t *testing.T) {
	logger := logr.Discard()
	validator := &ACSBillingValidator{
		Logger: logger,
	}

	tests := []struct {
		name        string
		key         string
		wantProduct string
		wantWindow  string
		wantErr     bool
	}{
		{
			name:        "valid key with today",
			key:         "ecs:today",
			wantProduct: "ecs",
			wantWindow:  "today",
			wantErr:     false,
		},
		{
			name:        "valid key with yesterday",
			key:         "rds:yesterday",
			wantProduct: "rds",
			wantWindow:  "yesterday",
			wantErr:     false,
		},
		{
			name:        "valid key with thisMonth",
			key:         "oss:thisMonth",
			wantProduct: "oss",
			wantWindow:  "thisMonth",
			wantErr:     false,
		},
		{
			name:    "invalid key - missing timeWindow",
			key:     "ecs",
			wantErr: true,
		},
		{
			name:    "invalid key - empty productCode",
			key:     ":today",
			wantErr: true,
		},
		{
			name:    "invalid key - unsupported timeWindow",
			key:     "ecs:tomorrow",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := validator.parseKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if params.ProductCode != tt.wantProduct {
					t.Errorf("parseKey() ProductCode = %v, want %v", params.ProductCode, tt.wantProduct)
				}
				if params.TimeWindow != tt.wantWindow {
					t.Errorf("parseKey() TimeWindow = %v, want %v", params.TimeWindow, tt.wantWindow)
				}
			}
		})
	}
}

func TestAlibabaCloudBillingValidator_BuildResponse(t *testing.T) {
	logger := logr.Discard()
	validator := &AlibabaCloudBillingValidator{
		Logger: logger,
	}

	// Mock 账单结果
	mockResult := &utils.BillingQueryResult{
		Items: []utils.BillingItem{
			{
				InstanceID:   "i-xxxxx",
				ProductCode:  "ecs",
				ProductName:  "云服务器ECS",
				BillingDate:  "2025-11-21",
				PretaxAmount: 100.0,
				Currency:     "CNY",
			},
			{
				InstanceID:   "i-yyyyy",
				ProductCode:  "ecs",
				ProductName:  "云服务器ECS",
				BillingDate:  "2025-11-21",
				PretaxAmount: 200.0,
				Currency:     "CNY",
			},
		},
		TotalCount: 2,
	}

	params := utils.BillingQueryParams{
		ProductCode: "ecs",
		TimeWindow:  "today",
	}

	response := validator.buildResponse(mockResult, params)

	// 验证响应
	if response["productCode"] != "ecs" {
		t.Errorf("buildResponse() productCode = %v, want ecs", response["productCode"])
	}
	if response["timeWindow"] != "today" {
		t.Errorf("buildResponse() timeWindow = %v, want today", response["timeWindow"])
	}
	if response["totalPaymentAmount"] != 300.0 {
		t.Errorf("buildResponse() totalPaymentAmount = %v, want 300.0", response["totalPaymentAmount"])
	}
	if response["itemCount"] != 2 {
		t.Errorf("buildResponse() itemCount = %v, want 2", response["itemCount"])
	}
}

func TestAlibabaCloudBillingValidator_Validate_InvalidKey(t *testing.T) {
	logger := logr.Discard()

	// 注意：这里不会真正创建客户端，因为我们只测试 key 解析
	validator := &AlibabaCloudBillingValidator{
		Logger: logger,
	}

	// 构建测试请求
	providerReq := externaldata.ProviderRequest{
		APIVersion: "externaldata.gatekeeper.sh/v1alpha1",
		Kind:       "ProviderRequest",
		Request: externaldata.Request{
			Keys: []string{"invalid-key"},
		},
	}

	reqBody, _ := json.Marshal(providerReq)
	req := httptest.NewRequest(http.MethodPost, "/validate/alibabacloud-billing", strings.NewReader(string(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	validator.Validate(w, req)

	// 验证响应
	var response externaldata.ProviderResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Response.Items) != 1 {
		t.Errorf("expected 1 item in response, got %d", len(response.Response.Items))
	}

	if response.Response.Items[0].Error == "" {
		t.Errorf("expected error in response for invalid key")
	}
}
