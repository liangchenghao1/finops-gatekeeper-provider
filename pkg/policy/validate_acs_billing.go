package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/config"
	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/utils"
	"github.com/go-logr/logr"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
	"golang.org/x/sync/singleflight"
	"net/http"
)

type billingCache struct {
	data map[string]*cacheEntry
	mu   sync.RWMutex
}

type cacheEntry struct {
	result    *utils.BillingQueryResult
	createdAt time.Time
}

type ACSBillingValidator struct {
	billingClient *utils.AlibabaCloudBillingClient
	clusterID     string
	cache         *billingCache
	sfGroup       singleflight.Group
	Logger        logr.Logger
}

func NewACSBillingValidator(logger logr.Logger, config *config.Config) *ACSBillingValidator {
	accessKeyID := os.Getenv("ALIBABACLOUD_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ALIBABACLOUD_ACCESS_KEY_SECRET")
	endpoint := os.Getenv("ALIBABACLOUD_ENDPOINT")

	if accessKeyID == "" || accessKeySecret == "" {
		logger.Info("alibabacloud credentials not configured, ACSBillingValidator will be disabled")
		return &ACSBillingValidator{Logger: logger, billingClient: nil, clusterID: config.Cluster.ClusterID}
	}

	billingClient, err := utils.NewAlibabaCloudBillingClient(utils.AlibabaCloudBillingConfig{
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		Endpoint:        endpoint,
	}, logger)
	if err != nil {
		logger.Error(err, "failed to create alibabacloud billing client")
		return &ACSBillingValidator{Logger: logger, billingClient: nil, clusterID: config.Cluster.ClusterID}
	}

	v := &ACSBillingValidator{
		Logger:        logger,
		billingClient: billingClient,
		clusterID:     config.Cluster.ClusterID,
		cache:         &billingCache{data: make(map[string]*cacheEntry)},
	}

	// 每小时清理过期缓存
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			v.cache.mu.Lock()
			for key, entry := range v.cache.data {
				if time.Since(entry.createdAt) >= 1*time.Hour {
					delete(v.cache.data, key)
				}
			}
			v.cache.mu.Unlock()
		}
	}()

	return v
}

func (v *ACSBillingValidator) Validate(w http.ResponseWriter, req *http.Request) {
	if v.billingClient == nil {
		v.Logger.Error(fmt.Errorf("alibabacloud billing client not initialized"), "credentials not configured")
		utils.SendResponse(w, nil, "AlibabaCloud credentials not configured")
		return
	}

	providerRequest, err := utils.ReadProviderRequest(w, req)
	if err != nil {
		v.Logger.Error(err, "failed to read provider request")
		return
	}

	results := make([]externaldata.Item, 0)
	for _, key := range providerRequest.Request.Keys {
		// key 格式: "timeWindow" 或 "timeWindow:namespace"
		parts := strings.Split(key, ":")
		timeWindow := parts[0]
		namespace := ""
		if len(parts) > 1 {
			namespace = parts[1]
		}

		// 获取账单数据（带缓存）
		result, err := v.getBillingData(timeWindow, namespace)
		if err != nil {
			v.Logger.Error(err, "failed to get billing data", "key", key)
			results = append(results, externaldata.Item{
				Key:   key,
				Error: err.Error(),
			})
			continue
		}

		var totalAmount float64
		var currency string
		for _, item := range result.Items {
			totalAmount += item.PretaxAmount
			if currency == "" && item.Currency != "" {
				currency = item.Currency
			}
		}

		if currency == "" {
			currency = "CNY" // 默认值
		}

		responseData := map[string]interface{}{
			"productCode": "acs",
			"timeWindow":  timeWindow,
			"namespace":   namespace,
			"clusterID":   v.clusterID,
			"totalAmount": totalAmount,
			"itemCount":   len(result.Items),
			"currency":    currency,
		}
		responseJSON, _ := json.Marshal(responseData)

		results = append(results, externaldata.Item{
			Key:   key,
			Value: string(responseJSON),
		})
	}

	utils.SendResponse(w, &results, "")
}

func (v *ACSBillingValidator) getBillingData(timeWindow, namespace string) (*utils.BillingQueryResult, error) {
	cacheKey := fmt.Sprintf("%s:%s", v.clusterID, timeWindow)
	v.Logger.Info("[getBillingData] request received", "cacheKey", cacheKey, "namespace", namespace)

	// 快速检查缓存
	v.cache.mu.RLock()
	entry, exists := v.cache.data[cacheKey]
	v.cache.mu.RUnlock()

	if exists && time.Since(entry.createdAt) < 1*time.Hour {
		v.Logger.Info("[getBillingData] cache hit", "cacheKey", cacheKey, "age", time.Since(entry.createdAt))
		return v.filterByNamespace(entry.result, namespace), nil
	}

	v.Logger.Info("[getBillingData] cache miss or expired, entering singleflight", "cacheKey", cacheKey)

	// 使用 singleflight 防止并发重复请求
	val, err, shared := v.sfGroup.Do(cacheKey, func() (interface{}, error) {
		v.Logger.Info("[singleflight] executing function", "cacheKey", cacheKey)

		// 再次检查缓存（可能已被其他协程更新）
		v.cache.mu.RLock()
		entry, exists := v.cache.data[cacheKey]
		v.cache.mu.RUnlock()

		if exists && time.Since(entry.createdAt) < 1*time.Hour {
			v.Logger.Info("[singleflight] cache hit on double-check", "cacheKey", cacheKey)
			return entry.result, nil
		}

		// 查询API
		v.Logger.Info("[singleflight] calling API", "cacheKey", cacheKey)
		result, err := v.billingClient.QueryInstanceBilling(utils.BillingQueryParams{
			ProductCode: "acs",
			TimeWindow:  timeWindow,
			ClusterID:   v.clusterID,
		})
		if err != nil {
			v.Logger.Error(err, "[singleflight] API call failed", "cacheKey", cacheKey)
			return nil, err
		}

		v.Logger.Info("[singleflight] API call success, updating cache", "cacheKey", cacheKey, "itemCount", len(result.Items))

		// 更新缓存
		v.cache.mu.Lock()
		v.cache.data[cacheKey] = &cacheEntry{
			result:    result,
			createdAt: time.Now(),
		}
		v.cache.mu.Unlock()

		return result, nil
	})

	if err != nil {
		return nil, err
	}

	if shared {
		v.Logger.Info("[getBillingData] result shared from singleflight", "cacheKey", cacheKey)
	} else {
		v.Logger.Info("[getBillingData] result from direct execution", "cacheKey", cacheKey)
	}

	return v.filterByNamespace(val.(*utils.BillingQueryResult), namespace), nil
}

func (v *ACSBillingValidator) filterByNamespace(result *utils.BillingQueryResult, namespace string) *utils.BillingQueryResult {
	if namespace == "" {
		return result
	}

	filteredItems := make([]utils.BillingItem, 0)
	for _, item := range result.Items {
		if item.Tags["NameSpace"] == namespace {
			filteredItems = append(filteredItems, item)
		}
	}
	return &utils.BillingQueryResult{
		Items:      filteredItems,
		TotalCount: int32(len(filteredItems)),
	}
}
