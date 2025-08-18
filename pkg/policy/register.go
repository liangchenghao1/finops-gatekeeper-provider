package policy

import (
	"fmt"
	"strings"

	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/config"
	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/server"
	"github.com/go-logr/logr"
)

var (
	// 全局变量存储注册的webhook处理器
	defaultResourcesMutator              *DefaultResourcesMutator
	defaultLabelsMutator                 *DefaultLabelsMutator
	resourcesValidator                   *ResourcesValidator
	labelsValidator                      *LabelsValidator
	workloadBudgetValidator              *WorkloadBudgetValidator
	workloadResourceUtilizationValidator *WorkloadResourceUtilizationValidator
	workloadRecommendationValidator      *WorkloadRecommendationValidator
)

// RegisterWebhooks 注册所有策略webhook
func RegisterWebhooks(logger logr.Logger, config *config.Config) {
	defaultResourcesMutator = NewDefaultResourcesMutator(logger)
	defaultLabelsMutator = NewDefaultLabelsMutator(logger)
	resourcesValidator = NewResourcesValidator(logger)
	labelsValidator = NewLabelsValidator(logger)
	workloadBudgetValidator = NewWorkloadBudgetValidator(logger)
	workloadResourceUtilizationValidator = NewWorkloadResourceUtilizationValidator(logger, config)
	workloadRecommendationValidator = NewWorkloadRecommendationValidator(logger)
}

// GetDefaultResourcesMutator 获取默认资源mutator
func GetDefaultResourcesMutator() server.MutatingWebhook {
	return defaultResourcesMutator
}

// GetDefaultLabelsMutator 获取默认标签mutator
func GetDefaultLabelsMutator() server.MutatingWebhook {
	return defaultLabelsMutator
}

// GetResourcesValidator 获取资源validator
func GetResourcesValidator() server.ValidatingWebhook {
	return resourcesValidator
}

// GetLabelsValidator 获取标签validator
func GetLabelsValidator() server.ValidatingWebhook {
	return labelsValidator
}

// RegisterWebhooksFromConfig 根据配置文件注册webhook
func RegisterWebhooksFromConfig(registry server.WebhookRegistry, cfg *config.Config, logger logr.Logger) error {
	// 初始化所有webhook处理器
	RegisterWebhooks(logger, cfg)

	// 遍历配置中的webhook并注册
	for _, webhookCfg := range cfg.Webhooks {
		// 跳过禁用的webhook
		if !webhookCfg.Enabled {
			logger.Info("skipping disabled webhook", "name", webhookCfg.Name, "path", webhookCfg.Path)
			continue
		}

		// 根据webhook名称和类型确定处理器
		var err error
		switch {
		case webhookCfg.Type == "mutating" && strings.Contains(webhookCfg.Name, "resources"):
			err = registry.RegisterMutatingWebhook(webhookCfg.Path, GetDefaultResourcesMutator())
		case webhookCfg.Type == "mutating" && strings.Contains(webhookCfg.Name, "labels"):
			err = registry.RegisterMutatingWebhook(webhookCfg.Path, GetDefaultLabelsMutator())
		case webhookCfg.Type == "validating" && strings.Contains(webhookCfg.Name, "resources"):
			err = registry.RegisterValidatingWebhook(webhookCfg.Path, GetResourcesValidator())
		case webhookCfg.Type == "validating" && strings.Contains(webhookCfg.Name, "labels"):
			err = registry.RegisterValidatingWebhook(webhookCfg.Path, GetLabelsValidator())
		case webhookCfg.Type == "validating" && strings.Contains(webhookCfg.Name, "workload-budget"):
			err = registry.RegisterValidatingWebhook(webhookCfg.Path, workloadBudgetValidator)
		case webhookCfg.Type == "validating" && strings.Contains(webhookCfg.Name, "workload-resource-utilization"):
			err = registry.RegisterValidatingWebhook(webhookCfg.Path, workloadResourceUtilizationValidator)
		case webhookCfg.Type == "validating" && strings.Contains(webhookCfg.Name, "workload-recommendation"):
			err = registry.RegisterValidatingWebhook(webhookCfg.Path, workloadRecommendationValidator)

		default:
			return fmt.Errorf("unknown webhook type or name: %s (%s)", webhookCfg.Name, webhookCfg.Type)
		}

		if err != nil {
			return fmt.Errorf("failed to register webhook %s: %w", webhookCfg.Name, err)
		}

		logger.Info("registered webhook",
			"name", webhookCfg.Name,
			"path", webhookCfg.Path,
			"type", webhookCfg.Type)
	}

	return nil
}
