package utils

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
)

// K8sClient Kubernetes客户端包装
type K8sClient struct {
	clientset     kubernetes.Interface
	dynamicClient dynamic.Interface
}

func (k *K8sClient) GetDeployment(namespace, name string) (*appsv1.Deployment, error) {
	return k.clientset.AppsV1().Deployments(namespace).Get(
		context.TODO(),
		name,
		metav1.GetOptions{},
	)
}

func (k *K8sClient) GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error) {
	return k.clientset.AppsV1().StatefulSets(namespace).Get(
		context.TODO(),
		name,
		metav1.GetOptions{},
	)
}

func (k *K8sClient) GetDaemonSet(namespace, name string) (*appsv1.DaemonSet, error) {
	return k.clientset.AppsV1().DaemonSets(namespace).Get(
		context.TODO(),
		name,
		metav1.GetOptions{},
	)
}

// NewK8sClient 创建新的K8sClient实例
func NewK8sClient() (*K8sClient, error) {
	// 创建集群内配置
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Errorf("Failed to create in cluster config: %v", err)

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %v", err)
		}

		kubeconfig := filepath.Join(homeDir, ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes config from %s: %v", err, kubeconfig)
		}
	}

	// 创建clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %v", err)
	}

	return &K8sClient{
		clientset:     clientset,
		dynamicClient: dynamicClient,
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
