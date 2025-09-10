package policy

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"

	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/utils"
)

// DefaultLabelsMutator 为没有设置app标签的应用添加默认标签
type DefaultLabelsMutator struct {
	Logger logr.Logger
}

// NewDefaultLabelsMutator 创建一个新的DefaultLabelsMutator实例
func NewDefaultLabelsMutator(logger logr.Logger) *DefaultLabelsMutator {
	return &DefaultLabelsMutator{
		Logger: logger,
	}
}

// Mutate 实现默认标签设置逻辑
func (m *DefaultLabelsMutator) Mutate(w http.ResponseWriter, req *http.Request) {
	// 读取并解析请求
	providerRequest, err := utils.ReadProviderRequest(w, req)
	if err != nil {
		m.Logger.Error(err, "failed to read provider request")
		return
	}

	// 处理每个key（应用名称）
	results := make([]externaldata.Item, 0)
	for _, key := range providerRequest.Request.Keys {
		// 从key中提取应用名称，通常key可能是"namespace/name"格式
		appName := key
		if parts := strings.Split(key, "/"); len(parts) > 1 {
			appName = parts[1] // 取name部分
		}

		// 创建默认标签
		defaultLabels := map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels": map[string]interface{}{
					"app": appName,
				},
			},
		}

		value, _ := json.Marshal(defaultLabels)
		results = append(results, externaldata.Item{
			Key:   key,
			Value: string(value),
		})

		m.Logger.Info("Added default label for app", "app", appName)
	}

	utils.SendResponse(w, &results, "")
}
