package policy

import (
	"net/http"

	"github.com/go-logr/logr"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
	
	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/utls"
)

// LabelsValidator 验证应用是否设置了必要的标签
type LabelsValidator struct {
	Logger logr.Logger
}

// NewLabelsValidator 创建一个新的LabelsValidator实例
func NewLabelsValidator(logger logr.Logger) *LabelsValidator {
	return &LabelsValidator{
		Logger: logger,
	}
}

// Validate 验证应用是否包含必要的标签
func (v *LabelsValidator) Validate(w http.ResponseWriter, req *http.Request) {
	// 读取并解析请求
	providerRequest, err := utls.ReadProviderRequest(w, req)
	if err != nil {
		v.Logger.Error(err, "failed to read provider request")
		return
	}

	// 验证每个key（应用名称）
	results := make([]externaldata.Item, 0)
	for _, key := range providerRequest.Request.Keys {
		// 这里应该实现具体的验证逻辑，检查应用是否设置了app标签
		// 如果没有设置，应该拒绝该请求并发送Event到k8s集群
		
		// 示例实现：假设所有应用都通过验证
		validationResult := "valid"
		
		// 如果验证失败，可以设置为"invalid"并添加详细信息
		// validationResult = "Required label 'app' not specified for application " + key
		
		results = append(results, externaldata.Item{
			Key:   key,
			Value: validationResult,
		})
		
		v.Logger.Info("Validated labels for app", "app", key, "result", validationResult)
	}

	utls.SendResponse(w, &results, "")
}