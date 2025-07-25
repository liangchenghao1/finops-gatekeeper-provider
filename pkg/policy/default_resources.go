package policy

import (
	"encoding/json"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
	
	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/utls"
)

// DefaultResourcesMutator 为没有设置资源请求的Pod设置默认值
type DefaultResourcesMutator struct {
	Logger logr.Logger
}

// NewDefaultResourcesMutator 创建一个新的DefaultResourcesMutator实例
func NewDefaultResourcesMutator(logger logr.Logger) *DefaultResourcesMutator {
	return &DefaultResourcesMutator{
		Logger: logger,
	}
}

// Mutate 实现资源默认值设置逻辑
func (m *DefaultResourcesMutator) Mutate(w http.ResponseWriter, req *http.Request) {
	// 读取并解析请求
	providerRequest, err := utls.ReadProviderRequest(w, req)
	if err != nil {
		m.Logger.Error(err, "failed to read provider request")
		return
	}

	// 处理每个key（Pod名称）
	results := make([]externaldata.Item, 0)
	for _, key := range providerRequest.Request.Keys {
		// 为没有资源请求的Pod添加默认资源
		defaultResources := map[string]interface{}{
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{
					"cpu":    "100m",
					"memory": "128Mi",
				},
				"limits": map[string]interface{}{
					"cpu":    "200m",
					"memory": "256Mi",
				},
			},
		}
		
		value, _ := json.Marshal(defaultResources)
		results = append(results, externaldata.Item{
			Key:   key,
			Value: string(value),
		})
		
		m.Logger.Info("Added default resources for pod", "pod", key)
	}

	utls.SendResponse(w, &results, "")
}

// sendResponse 发送响应给Gatekeeper
func (m *DefaultResourcesMutator) sendResponse(results *[]externaldata.Item, systemErr string, w http.ResponseWriter) {
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