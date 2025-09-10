package policy

import (
	"context"
	"fmt"
	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/config"
	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/utils"
	"github.com/go-logr/logr"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
	"github.com/prometheus/common/model"
	"net/http"
	"time"
)

const (
	QueryCpuUtilization = `
		avg_over_time(
		  sum(
			label_replace(
			  max(kube_pod_info{namespace="%s",created_by_kind="ReplicaSet", pod_ip!=""}) by (created_by_name, uid, pod, pod_ip, node),
			  "replicaset",
			  "$1",
			  "created_by_name",
			  "(.+)"
			)
			* on(replicaset) group_left()
			max(kube_replicaset_owner{owner_kind="Deployment",owner_name="%s"}) by (replicaset)
			* on(pod) group_right()
			(
			  max(irate(container_cpu_usage_seconds_total{container!="",container!="POD"}[1m])) by (pod, container)
			  /
			  max by(container, pod) (kube_pod_container_resource_requests{resource="cpu"})
			)
		  )[24h:5m]
		)
		`
)

// WorkloadResourceUtilizationValidator 验证应用是否设置了必要的标签
type WorkloadResourceUtilizationValidator struct {
	promClient *utils.PrometheusClient
	Logger     logr.Logger
}

// NewWorkloadResourceUtilizationValidator WorkloadResourceUtilizationValidator
func NewWorkloadResourceUtilizationValidator(logger logr.Logger, config *config.Config) *WorkloadResourceUtilizationValidator {
	promClient, err := utils.NewPrometheusClient(config.Prometheus.URL)
	if err != nil {
		logger.Error(err, "failed to create prom client")
		return nil
	}

	return &WorkloadResourceUtilizationValidator{
		Logger:     logger,
		promClient: promClient,
	}
}

// Validate 验证应用CPU利用率
func (v *WorkloadResourceUtilizationValidator) Validate(w http.ResponseWriter, req *http.Request) {
	// 读取并解析请求
	providerRequest, err := utils.ReadProviderRequest(w, req)
	if err != nil {
		v.Logger.Error(err, "failed to read provider request")
		return
	}

	results := make([]externaldata.Item, 0)
	for _, key := range providerRequest.Request.Keys {
		_, namespace, controllerName, err := utils.ParseWorkloadKey(key)
		if err != nil {
			v.Logger.Error(err, "failed to parse workload key", "key", key)
			return
		}

		result, warn, err := v.promClient.Query(context.Background(), fmt.Sprintf(QueryCpuUtilization, namespace, controllerName), time.Now())

		// 处理警告
		if warn != nil {
			v.Logger.Info("received prometheus query warning", "warning", warn, "key", key)
		}

		// 处理查询错误
		if err != nil {
			v.Logger.Error(err, "prometheus query failed", "key", key)
			results = append(results, externaldata.Item{
				Key:   key,
				Error: err.Error(),
			})
			continue
		}

		samples, ok := result.(model.Vector)
		if !ok {
			errMsg := "unexpected result type from prometheus"
			v.Logger.Error(fmt.Errorf(errMsg), "type assertion failed", "result", result)
			results = append(results, externaldata.Item{
				Key:   key,
				Error: errMsg,
			})
			continue
		}

		// 处理空结果
		if len(samples) == 0 {
			results = append(results, externaldata.Item{
				Key:   key,
				Error: "no utilization data available",
			})
			v.Logger.Info("no utilization data found", "key", key)
			continue
		}

		// 提取第一个样本值（假设查询返回单个平均值）
		avgUtilization := float64(samples[0].Value) * 100 // 转换为百分比

		results = append(results, externaldata.Item{
			Key:   key,
			Value: fmt.Sprintf("%.2f%%", avgUtilization),
		})

		v.Logger.Info("Successfully queried utilization",
			"workload", key,
			"namespace", namespace,
			"controller", controllerName,
			"utilization", avgUtilization)
	}

	utils.SendResponse(w, &results, "")
}
