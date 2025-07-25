package utls

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// K8sClient Kubernetes客户端包装
type K8sClient struct {
	clientset kubernetes.Interface
}

// NewK8sClient 创建新的K8sClient实例
func NewK8sClient() (*K8sClient, error) {
	// 创建集群内配置
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %v", err)
	}

	// 创建clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	return &K8sClient{
		clientset: clientset,
	}, nil
}

// GetPod 获取指定命名空间和名称的Pod
func (k *K8sClient) GetPod(namespace, name string) (*corev1.Pod, error) {
	return k.clientset.CoreV1().Pods(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// CreateEvent 创建Kubernetes Event
func (k *K8sClient) CreateEvent(namespace, name, reason, message string, objectType string) error {
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Reason:  reason,
		Message: message,
		Type:    "Warning",
		InvolvedObject: corev1.ObjectReference{
			Namespace: namespace,
			Name:      name,
			Kind:      objectType,
		},
	}

	_, err := k.clientset.CoreV1().Events(namespace).Create(context.TODO(), event, metav1.CreateOptions{})
	return err
}