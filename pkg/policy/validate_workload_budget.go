package policy

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
	"net/http"
	"strings"

	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/utls"
)

// WorkloadBudgetValidator 验证应用是否设置了必要的标签
type WorkloadBudgetValidator struct {
	k8sClient *utls.K8sClient
	Logger    logr.Logger
}

// NewLabelsValidator 创建一个新的LabelsValidator实例
func NewWorkloadBudgetValidator(logger logr.Logger) *WorkloadBudgetValidator {
	k8sClient, err := utls.NewK8sClient()
	if err != nil {
		logger.Error(err, "failed to create k8s client")
		return nil
	}

	return &WorkloadBudgetValidator{
		Logger:    logger,
		k8sClient: k8sClient,
	}
}

// Validate 验证应用是否包含必要的标签
func (v *WorkloadBudgetValidator) Validate(w http.ResponseWriter, req *http.Request) {
	// 读取并解析请求
	providerRequest, err := utls.ReadProviderRequest(w, req)
	if err != nil {
		v.Logger.Error(err, "failed to read provider request")
		return
	}

	results := make([]externaldata.Item, 0)
	for _, key := range providerRequest.Request.Keys {
		namespace, controllerName, err := parseWorkloadKey(key)
		if err != nil {
			v.Logger.Error(err, "failed to parse workload key", "key", key)
			return
		}

		costResp, costErr := v.k8sClient.QueryCost(context.Background(), utls.CostQuery{
			Window: "yesterday",
			Filter: []utls.CostFilter{
				{Type: utls.FilterNamespace, Value: `"` + namespace + `"`},
				{Type: utls.FilterController, Value: `"` + controllerName + `"`},
			},
			Aggregate: "controller",
		})
		if costErr != nil {
			v.Logger.Error(costErr, "failed to query cost")
			return
		}

		if len(costResp.Allocations) != 1 {
			v.Logger.Error(costErr, "failed to query cost")
		}
		items := *costResp.Allocations[0]
		var cost float64
		for _, item := range items {
			cost = item.Cost
			break
		}

		results = append(results, externaldata.Item{
			Key:   key,
			Value: cost,
		})

		v.Logger.Info("Validated labels for app", "app", key, "result", cost)
	}

	utls.SendResponse(w, &results, "")
}

func parseWorkloadKey(key string) (namespace, workload string, err error) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid workload key %q, expect <namespace>/<name>", key)
	}
	return parts[0], parts[1], nil
}
