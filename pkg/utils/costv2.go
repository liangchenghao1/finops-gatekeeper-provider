package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// QueryCost 扩展您的K8sClient结构体添加成本查询方法
func (k *K8sClient) QueryCost(ctx context.Context, query CostQuery) (*AllocationSetRange, error) {
	// 构建请求
	req := k.clientset.CoreV1().RESTClient().Get().
		AbsPath("/api/v1/namespaces/kube-system/services/ack-metrics-adapter-api-service:8080/proxy/v2/cost").
		Param("window", query.Window)

	// 处理过滤条件
	if len(query.Filter) > 0 {
		req.Param("filter", buildFilterString(query.Filter))
	}

	// 添加可选参数
	if query.Step != "" {
		req.Param("step", query.Step)
	}
	if query.Aggregate != "" {
		req.Param("aggregate", query.Aggregate)
	}
	if query.ShareSplit != "" {
		req.Param("shareSplit", query.ShareSplit)
	}
	if query.Format != "" {
		req.Param("format", query.Format)
	}
	if query.Idle != nil {
		req.Param("idle", fmt.Sprintf("%t", *query.Idle))
	}
	if query.ShareIdle != nil {
		req.Param("shareIdle", fmt.Sprintf("%t", *query.ShareIdle))
	}
	if query.IdleByNode != nil {
		req.Param("idleByNode", fmt.Sprintf("%t", *query.IdleByNode))
	}

	// 执行请求
	// todo 增加更详细的错误透出如权限不足
	raw, err := req.DoRaw(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query cost: %v. query :%+v", err, query)
	}

	// 解析JSON响应
	var resp AllocationSetRange
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response to AllocationSetRange: %v", err)
	}
	return &resp, nil
}

// 直接在现有文件中定义相关类型
type CostQuery struct {
	Window     string
	Filter     []CostFilter
	Step       string
	Aggregate  string
	Idle       *bool
	ShareIdle  *bool
	ShareSplit string
	IdleByNode *bool
	Format     string // "json" 或 "csv"
}

type CostFilter struct {
	Type  CostFilterType
	Value string
}

type CostFilterType string

const (
	FilterNamespace      CostFilterType = "namespace"
	FilterController     CostFilterType = "controllerName"
	FilterControllerKind CostFilterType = "controllerKind"
	FilterPod            CostFilterType = "pod"
	FilterLabel          CostFilterType = "label"
)

type AllocationSetRange struct {
	Allocations []*AllocationSet `json:"data"`
}

type Allocation struct {
	Name       string                `json:"name"`
	Properties *AllocationProperties `json:"properties,omitempty"`
	//Window               *Window                `json:"window"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	//CPUCoreHours          float64   `json:"cpuCoreHours"`
	CPUCoreRequestAverage float64 `json:"cpuCoreRequestAverage"`
	CPUCoreUsageAverage   float64 `json:"cpuCoreUsageAverage"`
	//GPUHours               float64               `json:"gpuHours"`
	//RAMByteHours           float64 `json:"ramByteHours"`
	RAMBytesRequestAverage float64 `json:"ramByteRequestAverage"`
	RAMBytesUsageAverage   float64 `json:"ramByteUsageAverage"`
	Cost                   float64 `json:"cost"`
	CostRatio              float64 `json:"costRatio"`
	CustomCost             float64 `json:"customCost"`
}

type AllocationProperties struct {
	Cluster        string            `json:"cluster,omitempty"`
	Node           string            `json:"node,omitempty"`
	Controller     string            `json:"controller,omitempty"`
	ControllerKind string            `json:"controllerKind,omitempty"`
	Namespace      string            `json:"namespace,omitempty"`
	Pod            string            `json:"pod,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	ProviderID     string            `json:"providerID,omitempty"`
}

type AllocationSet map[string]*Allocation

// 辅助函数构建过滤字符串
func buildFilterString(filters []CostFilter) string {
	var parts []string
	for _, f := range filters {
		switch f.Type {
		case FilterLabel:
			if kv := strings.SplitN(f.Value, ":", 2); len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				parts = append(parts, fmt.Sprintf(`label[%s]:%s`, key, value))
			}
		default:
			parts = append(parts, fmt.Sprintf(`%s:%s`, f.Type, f.Value))
		}
	}
	return strings.Join(parts, "+")
}
