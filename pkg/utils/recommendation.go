package utils

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	LabelKeyRecommendationWorkloadName = "alpha.alibabacloud.com/recommendation-workload-name"
)

type ContainerTarget struct {
	CPU    string
	Memory string
}

type RecommendationTargets map[string]ContainerTarget

func (k *K8sClient) GetRecommendation(namespace, workloadName string) (RecommendationTargets, error) {
	recommendationGvr := schema.GroupVersionResource{
		Group:    "autoscaling.alibabacloud.com",
		Version:  "v1alpha1",
		Resource: "recommendations",
	}

	labelSelector := labels.Set{LabelKeyRecommendationWorkloadName: workloadName}.AsSelector()
	list, err := k.dynamicClient.Resource(recommendationGvr).Namespace(namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: labelSelector.String()},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list recommendations for %s: %w", workloadName, err)
	}
	fmt.Sprintf("%v", list)

	if len(list.Items) == 0 {
		return nil, fmt.Errorf("no recommendation found for workload %s", workloadName)
	}

	return extractContainerTargets(&list.Items[0])

}

// extractContainerTargets 从Unstructured对象中提取容器目标
func extractContainerTargets(rec *unstructured.Unstructured) (RecommendationTargets, error) {
	status, found, err := unstructured.NestedMap(rec.Object, "status")
	if err != nil {
		return nil, fmt.Errorf("failed to get status field: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("recommendation %s has no status field", rec.GetName())
	}

	recommendResources, found, err := unstructured.NestedMap(status, "recommendResources")
	if err != nil {
		return nil, fmt.Errorf("failed to get recommendResources: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("recommendation %s has no recommendResources", rec.GetName())
	}

	containerRecs, found, err := unstructured.NestedSlice(recommendResources, "containerRecommendations")
	if err != nil {
		return nil, fmt.Errorf("failed to get containerRecommendations: %w", err)
	}
	if !found || len(containerRecs) == 0 {
		return nil, fmt.Errorf("no container recommendations found")
	}

	targets := make(RecommendationTargets)
	for _, cr := range containerRecs {
		containerRec, ok := cr.(map[string]interface{})
		if !ok {
			continue
		}

		containerName, _, _ := unstructured.NestedString(containerRec, "containerName")
		if containerName == "" {
			continue
		}

		targetMap, found, _ := unstructured.NestedMap(containerRec, "target")
		if !found {
			continue
		}

		cpu, _, _ := unstructured.NestedString(targetMap, "cpu")
		memory, _, _ := unstructured.NestedString(targetMap, "memory")

		targets[containerName] = ContainerTarget{
			CPU:    cpu,
			Memory: memory,
		}
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("valid container targets not found")
	}

	return targets, nil
}
