package utls

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// PrometheusClient 封装Prometheus查询客户端
type PrometheusClient struct {
	api v1.API
}

// NewPrometheusClient 创建新的Prometheus客户端
func NewPrometheusClient(prometheusURL string) (*PrometheusClient, error) {
	client, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating prometheus client: %v", err)
	}

	return &PrometheusClient{
		api: v1.NewAPI(client),
	}, nil
}

// Query 执行Prometheus即时查询
func (p *PrometheusClient) Query(ctx context.Context, query string, timestamp time.Time) (model.Value, v1.Warnings, error) {
	return p.api.Query(ctx, query, timestamp)
}

// QueryRange 执行Prometheus范围查询
func (p *PrometheusClient) QueryRange(ctx context.Context, query string, r v1.Range) (model.Value, v1.Warnings, error) {
	return p.api.QueryRange(ctx, query, r)
}
