package policy

import (
	"net/http"

	"github.com/go-logr/logr"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
	
	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/utls"
)

// ResourcesValidator 验证Pod是否设置了必要的资源请求
type ResourcesValidator struct {
	Logger logr.Logger
}

// NewResourcesValidator 创建一个新的ResourcesValidator实例
func NewResourcesValidator(logger logr.Logger) *ResourcesValidator {
	return &ResourcesValidator{
		Logger: logger,
	}
}

// Validate 验证Pod是否包含必要的资源请求
func (v *ResourcesValidator) Validate(w http.ResponseWriter, req *http.Request) {
	// 读取并解析请求
	providerRequest, err := utls.ReadProviderRequest(w, req)
	if err != nil {
		v.Logger.Error(err, "failed to read provider request")
		return
	}

	// 验证每个key（Pod名称）
	results := make([]externaldata.Item, 0)
	for _, key := range providerRequest.Request.Keys {
		// 这里应该实现具体的验证逻辑，检查Pod是否设置了必要的资源请求
		// 如果没有设置，应该拒绝该请求并发送Event到k8s集群
		
		// 示例实现：假设所有Pod都通过验证
		validationResult := "valid"
		
		// 如果验证失败，可以设置为"invalid"并添加详细信息
		// validationResult = "Required resources (cpu and memory requests) not specified for pod " + key
		
		results = append(results, externaldata.Item{
			Key:   key,
			Value: validationResult,
		})
		
		v.Logger.Info("Validated resources for pod", "pod", key, "result", validationResult)
	}

	utls.SendResponse(w, &results, "")
}