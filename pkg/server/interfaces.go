package server

import (
	"net/http"
)

// MutatingWebhook 定义了Mutating Webhook的接口
type MutatingWebhook interface {
	// Mutate 处理mutating webhook请求
	Mutate(w http.ResponseWriter, r *http.Request)
}

// ValidatingWebhook 定义了Validating Webhook的接口
type ValidatingWebhook interface {
	// Validate 处理validating webhook请求
	Validate(w http.ResponseWriter, r *http.Request)
}

// WebhookRegistry 定义了Webhook注册接口
type WebhookRegistry interface {
	// RegisterMutatingWebhook 注册Mutating Webhook
	RegisterMutatingWebhook(path string, webhook MutatingWebhook) error
	// RegisterValidatingWebhook 注册Validating Webhook
	RegisterValidatingWebhook(path string, webhook ValidatingWebhook) error
}