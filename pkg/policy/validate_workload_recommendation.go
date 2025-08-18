package policy

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	"net/http"
	"strings"

	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/utls"
	"github.com/go-logr/logr"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
	corev1 "k8s.io/api/core/v1"
)

type WorkloadRecommendationValidator struct {
	k8sClient *utls.K8sClient
	logger    logr.Logger
}

// 新的请求数据结构（简化为资源标识）
type ResourceIdentifier struct {
	Namespace string
	Kind      string
	Name      string
}

const (
	cpuDiffThreshold    = 20
	memoryDiffThreshold = 32
)

func NewWorkloadRecommendationValidator(logger logr.Logger) *WorkloadRecommendationValidator {
	k8sClient, err := utls.NewK8sClient()
	if err != nil {
		logger.Error(err, "failed to create k8s client")
		return nil
	}

	return &WorkloadRecommendationValidator{
		logger:    logger.WithName("WorkloadRecommendationValidator"),
		k8sClient: k8sClient,
	}
}

func (v *WorkloadRecommendationValidator) Validate(w http.ResponseWriter, req *http.Request) {
	providerRequest, err := utls.ReadProviderRequest(w, req)
	if err != nil {
		v.logger.Error(err, "failed to read provider request")
		utls.SendResponse(w, nil, "invalid request format")
		return
	}

	results := make([]externaldata.Item, 0, len(providerRequest.Request.Keys))

	for _, key := range providerRequest.Request.Keys {
		kind, namespace, controllerName, err := utls.ParseWorkloadKey(key)

		resourceID := ResourceIdentifier{
			Namespace: namespace,
			Name:      controllerName,
			Kind:      kind,
		}

		// 获取实际容器配置
		actualContainers, err := v.getActualContainers(&resourceID)
		if err != nil {
			v.logger.Error(err, "failed to get container configuration",
				"namespace", resourceID.Namespace,
				"name", resourceID.Name)
			results = append(results, externaldata.Item{
				Key:   key,
				Error: "failed to get resource",
			})
			continue
		}

		// 获取推荐配置
		targets, err := v.k8sClient.GetRecommendation(resourceID.Namespace, resourceID.Name)
		v.logger.Info("get recommendation", "targets", targets)
		if err != nil {
			v.logger.Error(err, "failed to get recommendation",
				"namespace", resourceID.Namespace,
				"name", resourceID.Name)
			results = append(results, externaldata.Item{
				Key:   key,
				Error: "failed to get recommendation",
			})
			continue
		}

		var violations []string
		for _, container := range actualContainers {
			violations = append(violations, v.validateContainer(container, targets)...)
		}

		if len(violations) > 0 {
			results = append(results, externaldata.Item{
				Key:   key,
				Error: strings.Join(violations, "; "),
			})
		} else {
			results = append(results, externaldata.Item{
				Key:   key,
				Value: "valid",
			})
		}
	}

	utls.SendResponse(w, &results, "")
}

func (v *WorkloadRecommendationValidator) getActualContainers(resource *ResourceIdentifier) ([]corev1.Container, error) {
	switch strings.ToLower(resource.Kind) {
	case "deployment":
		deploy, err := v.k8sClient.GetDeployment(resource.Namespace, resource.Name)
		if err != nil {
			return nil, err
		}
		return deploy.Spec.Template.Spec.Containers, nil
	case "statefulset":
		sts, err := v.k8sClient.GetStatefulSet(resource.Namespace, resource.Name)
		if err != nil {
			return nil, err
		}
		return sts.Spec.Template.Spec.Containers, nil
	case "daemonset":
		ds, err := v.k8sClient.GetDaemonSet(resource.Namespace, resource.Name)
		if err != nil {
			return nil, err
		}
		return ds.Spec.Template.Spec.Containers, nil
	default:
		return nil, fmt.Errorf("unsupported workload type: %s", resource.Kind)
	}
}

// 调整后的校验方法
func (v *WorkloadRecommendationValidator) validateContainer(
	container corev1.Container,
	targets utls.RecommendationTargets,
) []string {
	var violations []string

	target, exists := targets[container.Name]
	if !exists {
		return []string{fmt.Sprintf("%s: no recommendation found", container.Name)}
	}

	// 验证Requests
	if requests := container.Resources.Requests; len(requests) > 0 {
		v.checkResources("request", container.Name, requests, target, &violations)
	} else {
		violations = append(violations, fmt.Sprintf("%s: requests not set", container.Name))
	}

	return violations
}

func (v *WorkloadRecommendationValidator) checkResources(
	resType string,
	containerName string,
	actual corev1.ResourceList,
	target utls.ContainerTarget,
	violations *[]string,
) {
	if cpu, ok := actual[corev1.ResourceCPU]; ok {
		if diff, err := compareCPU(cpu, target.CPU); err != nil {
			*violations = append(*violations,
				fmt.Sprintf("%s: cpu %s error - %v", containerName, resType, err))
		} else if diff > cpuDiffThreshold {
			*violations = append(*violations,
				fmt.Sprintf("%s: the diff between cpu %s and recommendation(%s) is %dm, threshold is %dm",
					containerName, resType, target.CPU, diff, cpuDiffThreshold))
		}
	}

	if memory, ok := actual[corev1.ResourceMemory]; ok {
		if diff, err := compareMemory(memory, target.Memory); err != nil {
			*violations = append(*violations,
				fmt.Sprintf("%s: memory %s error - %v", containerName, resType, err))
		} else if diff > memoryDiffThreshold {
			*violations = append(*violations,
				fmt.Sprintf("%s: the diff between memory %s and recommendation(%s) is %dMi, threshold is %dMi",
					containerName, resType, target.Memory, diff, memoryDiffThreshold))
		}
	}
}

func compareCPU(actual resource.Quantity, recommended string) (int64, error) {
	recQ, err := resource.ParseQuantity(recommended)
	if err != nil {
		return 0, fmt.Errorf("invalid recommendation format: %v", err)
	}

	actualMilli := actual.MilliValue()
	recMilli := recQ.MilliValue()

	return abs(actualMilli - recMilli), nil
}

func compareMemory(actual resource.Quantity, recommended string) (int64, error) {
	recQ, err := resource.ParseQuantity(recommended)
	if err != nil {
		return 0, fmt.Errorf("invalid recommendation format: %v", err)
	}

	actualMi := actual.Value() / 1024 / 1024
	recMi := recQ.Value() / 1024 / 1024

	return abs(actualMi - recMi), nil
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
