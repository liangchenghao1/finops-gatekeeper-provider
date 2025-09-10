# FinOps Gatekeeper Provider

FinOps Gatekeeper Provider 为 Open Policy Agent Gatekeeper 提供策略模版和外部数据服务，用于 FinOps 场景的 Kubernetes 资源管理和主动成本控制。

## 功能特性

### 核心功能
- **外部数据提供**: 为 Gatekeeper 提供实时成本、资源利用率等外部数据
- **多策略支持**: 支持 Mutating 和 Validating 策略
- **动态配置**: 可通过配置文件动态注册多个策略处理器

### FinOps 策略模板
- **工作负载预算控制**: 防止工作负载扩容时超出成本预算
- **资源利用率验证**: 基于实际资源利用率控制扩容操作
- **智能推荐验证**: 验证资源推荐配置的有效性
- **默认资源配置**: 自动为容器设置合理的资源请求和限制
- **标签管理**: 自动添加和管理 FinOps 相关标签
- **更多策略补充中**

## 快速开始

### 构建

```bash
# 拉取代码
git clone https://github.com/AliyunContainerService/finops-gatekeeper-provider.git
cd finops-gatekeeper-provider

# 或使用 Docker 构建
docker build -t finops-gatekeeper-provider:v0.1 .
```


### 部署


1. **生成证书**:
```bash
cd finops-gatekeeper-provider
bash ./scripts/generate-tls-cert.sh
```

2. **创建证书secret**:
```bash
kubectl create secret tls finops-external-data-secret -nkube-system --cert="./certs/tls.crt" --key="./certs/tls.key"
```

3. **创建 RBAC 配置**:
```bash
kubectl apply -f manifest/rbac.yaml
```

4. **创建 ConfigMap**:
```bash
# 配置注册的external data server
kubectl apply -f manifest/configmap.yaml
```

5. **部署服务**:
```bash
# 修改 manifest/deployment.yaml 为重新构建的镜像
kubectl apply -f manifest/deployment.yaml
kubectl apply -f manifest/service.yaml
```

4. **应用策略模板**:
```bash
# 应用所有策略模板
kubectl apply -f policies/validating/workload-budget/
kubectl apply -f policies/validating/workload-utilization/
kubectl apply -f policies/validating/recommendation/
```


## 配置

### 配置文件格式

External Data Server 支持通过 YAML 配置文件进行配置，默认查找 configmap 中的`config.yaml` 文件：

```yaml
server:
  port: 8090           # 服务端口
  timeout: 300         # 请求超时时间(秒)

webhooks:
  - name: "default-resources-mutator"
    path: "/mutate/resources"
    type: "mutating"
    enabled: true
  
  - name: "default-labels-mutator"
    path: "/mutate/labels"
    type: "mutating"
    enabled: true
  
  - name: "resources-validator"
    path: "/validate/resources"
    type: "validating"
    enabled: true
  
  - name: "labels-validator"
    path: "/validate/labels"
    type: "validating"
    enabled: true

  - name: "workload-budget-validator"
    path: "/validate/workload-budget"
    type: "validating"
    enabled: true

  - name: "workload-resource-utilization-validator"
    path: "/validate/workload-resource-utilization"
    type: "validating"
    enabled: true

  - name: "workload-recommendation-validator"
    path: "/validate/workload-recommendation"
    type: "validating"
    enabled: true

prometheus:
  url: "http://your-prometheus-url:9090/api/v1/query"
```

### 策略处理器映射

策略处理器根据名称自动映射：

| 策略名称模式 | 类型 | 功能描述 |
|-------------|------|----------|
| `*resources*` | mutating | 默认资源配置 Mutator |
| `*labels*` | mutating | 默认标签配置 Mutator |
| `*resources*` | validating | 资源配置 Validator |
| `*labels*` | validating | 标签配置 Validator |
| `*workload-budget*` | validating | 工作负载预算验证器 |
| `*workload-resource-utilization*` | validating | 资源利用率验证器 |
| `*workload-recommendation*` | validating | 工作负载推荐验证器 |

## FinOps 策略模板

### 1. 工作负载预算控制 (WorkloadBudget)

防止工作负载扩容时超出成本预算：

```yaml
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: WorkloadBudget
metadata:
  name: workload-budget-example
spec:
  match:
    kinds:
      - apiGroups: ["apps"]
        kinds: ["Deployment"]
  parameters:
    budget: 0.1  # 预算阈值
```

### 2. 资源利用率验证 (WorkloadUtilization)

基于实际资源利用率控制扩容操作：

```yaml
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: WorkloadUtilization
metadata:
  name: workload-utilization-example
spec:
  match:
    kinds:
      - apiGroups: ["apps"]
        kinds: ["Deployment"]
  parameters:
    threshold: 0.6  # 利用率阈值
```

### 3. 工作负载推荐验证 (WorkloadRecommendation)

验证资源推荐配置的有效性：

```yaml
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: WorkloadRecommendation
metadata:
  name: workload-recommendation-example
spec:
  match:
    kinds:
      - apiGroups: ["autoscaling"]
        kinds: ["HorizontalPodAutoscaler"]
```

## 开发

### 项目结构

```
├── main.go                    # 主程序入口
├── config.yaml               # 默认配置文件
├── pkg/
│   ├── config/               # 配置管理
│   ├── policy/               # 策略实现
│   ├── server/               # Webhook 服务器
│   └── utils/                # 工具函数
├── policies/                 # 策略模板
│   ├── mutating/             # 变更策略
│   └── validating/           # 验证策略
└── manifest/                 # Kubernetes 部署文件
```

### 添加新的策略处理器

1. 在 `pkg/policy/` 中实现新的处理器
2. 在 `register.go` 中注册处理器
3. 更新配置文件添加新的策略配置
4. 创建对应的策略模板（如需要）

## 许可证

本项目采用 Apache 2.0 许可证。