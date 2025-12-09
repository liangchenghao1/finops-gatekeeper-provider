package utils

import (
	"fmt"
	"strings"
	"time"

	bssopenapi "github.com/alibabacloud-go/bssopenapi-20171214/v4/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/go-logr/logr"
)

type AlibabaCloudBillingClient struct {
	client *bssopenapi.Client
	logger logr.Logger
}

type AlibabaCloudBillingConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	Endpoint        string
}

type BillingQueryParams struct {
	ProductCode string
	TimeWindow  string
	ClusterID   string
}

type BillingItem struct {
	InstanceID   string
	ProductCode  string
	ProductName  string
	BillingDate  string
	PretaxAmount float64
	Currency     string
	Tags         map[string]string
}

type BillingQueryResult struct {
	Items      []BillingItem
	TotalCount int32
}

func NewAlibabaCloudBillingClient(config AlibabaCloudBillingConfig, logger logr.Logger) (*AlibabaCloudBillingClient, error) {
	if config.AccessKeyID == "" || config.AccessKeySecret == "" {
		return nil, fmt.Errorf("AccessKeyID and AccessKeySecret are required")
	}

	if config.Endpoint == "" {
		config.Endpoint = "business.aliyuncs.com"
	}

	openApiConfig := &openapi.Config{
		AccessKeyId:     tea.String(config.AccessKeyID),
		AccessKeySecret: tea.String(config.AccessKeySecret),
		Endpoint:        tea.String(config.Endpoint),
	}

	client, err := bssopenapi.NewClient(openApiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create alibabacloud billing client: %v", err)
	}

	return &AlibabaCloudBillingClient{
		client: client,
		logger: logger,
	}, nil
}

func (c *AlibabaCloudBillingClient) QueryInstanceBilling(params BillingQueryParams) (*BillingQueryResult, error) {
	now := time.Now()
	var billingCycle, billingDate string

	switch params.TimeWindow {
	case "today":
		billingCycle = now.Format("2006-01")
		billingDate = now.Format("2006-01-02")
	case "yesterday":
		yesterday := now.AddDate(0, 0, -1)
		billingCycle = yesterday.Format("2006-01")
		billingDate = yesterday.Format("2006-01-02")
	default:
		return nil, fmt.Errorf("unsupported time window: %s", params.TimeWindow)
	}

	result := &BillingQueryResult{
		Items: make([]BillingItem, 0),
	}

	var nextToken *string

	// 打印整体请求参数
	c.logger.V(1).Info("[Billing API] calling DescribeInstanceBill",
		"billingCycle", billingCycle,
		"billingDate", billingDate,
		"productCode", params.ProductCode,
		"clusterID", params.ClusterID,
	)

	for {
		request := &bssopenapi.DescribeInstanceBillRequest{
			BillingCycle: tea.String(billingCycle),
			BillingDate:  tea.String(billingDate),
			ProductCode:  tea.String(params.ProductCode),
			Granularity:  tea.String("DAILY"),
			MaxResults:   tea.Int32(300),
			NextToken:    nextToken,
		}

		response, err := c.client.DescribeInstanceBill(request)
		if err != nil {
			c.logger.Error(err, "[Billing API] DescribeInstanceBill failed")
			return nil, fmt.Errorf("failed to query instance billing: %v", err)
		}

		if response.Body == nil || response.Body.Data == nil {
			break
		}

		if result.TotalCount == 0 {
			result.TotalCount = tea.Int32Value(response.Body.Data.TotalCount)
		}

		if response.Body.Data.Items != nil {
			for _, item := range response.Body.Data.Items {
				var pretaxAmount float64
				if item.PretaxAmount != nil {
					pretaxAmount = float64(*item.PretaxAmount)
				}

				tags := parseTags(tea.StringValue(item.Tag))

				// 根据 clusterID 过滤
				if params.ClusterID != "" && tags["acs:acc:cluster_id"] != params.ClusterID {
					continue
				}

				result.Items = append(result.Items, BillingItem{
					InstanceID:   tea.StringValue(item.InstanceID),
					ProductCode:  tea.StringValue(item.ProductCode),
					ProductName:  tea.StringValue(item.ProductName),
					BillingDate:  tea.StringValue(item.BillingDate),
					PretaxAmount: pretaxAmount,
					Currency:     tea.StringValue(item.Currency),
					Tags:         tags,
				})
			}
		}

		nextToken = response.Body.Data.NextToken
		if nextToken == nil || *nextToken == "" {
			break
		}
	}

	// 计算总金额
	var totalAmount float64
	for _, item := range result.Items {
		totalAmount += item.PretaxAmount
	}

	// 打印关键结果信息
	c.logger.Info("[Billing API] query completed",
		"itemCount", len(result.Items),
		"totalAmount", totalAmount,
	)

	return result, nil
}

func parseTags(tagStr string) map[string]string {
	tags := make(map[string]string)
	if tagStr == "" {
		return tags
	}

	pairs := strings.Split(tagStr, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, " value:", 2)
		if len(parts) == 2 {
			key := strings.TrimPrefix(strings.TrimSpace(parts[0]), "key:")
			value := strings.TrimSpace(parts[1])
			tags[key] = value
		}
	}
	return tags
}
