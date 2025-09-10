package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"
)

func Test(t *testing.T) {
	// 创建客户端（使用您现有的实现）
	client, err := NewK8sClient()
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	// 示例1: 查询DaemonSet昨日成本
	daemonsetCost, err := client.QueryCost(context.Background(), CostQuery{
		Window: "yesterday",
		Filter: []CostFilter{
			{Type: FilterNamespace, Value: `"kube-system"`},
			{Type: FilterController, Value: `"terway-eniip"`},
		},
		Aggregate: "controller",
	})
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}
	//printCosts("DaemonSet昨日成本", daemonsetCost)
	log.Printf("DaemonSet昨日成本: %+v", prettyPrint(daemonsetCost.Allocations))

	sss := *daemonsetCost.Allocations[0]

	log.Printf("CostResponse: %+v", sss["daemonset:terway-eniip"].Cost)
}

func prettyPrint(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return string(b)
}
