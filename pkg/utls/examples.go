package utls

import (
	"fmt"

	"github.com/go-logr/logr"
)

// ExampleK8sUsage 展示如何使用K8sClient的示例函数
func ExampleK8sUsage(logger logr.Logger) {
	// 创建K8sClient实例
	client, err := NewK8sClient()
	if err != nil {
		logger.Error(err, "Failed to create K8s client")
		return
	}
	
	// 确保变量被使用
	_ = client

	// 示例：获取Pod信息（需要提供实际的namespace和name）
	// pod, err := client.GetPod("default", "example-pod")
	// if err != nil {
	// 	logger.Error(err, "Failed to get pod")
	// 	return
	// }
	// logger.Info("Got pod", "pod", pod.Name, "namespace", pod.Namespace)

	// 示例：创建Event（需要提供实际的参数）
	// err = client.CreateEvent("default", "example-event", "ExampleReason", "This is an example event", "Pod")
	// if err != nil {
	// 	logger.Error(err, "Failed to create event")
	// 	return
	// }
	// logger.Info("Created event successfully")

	fmt.Println("K8s client usage example")
}