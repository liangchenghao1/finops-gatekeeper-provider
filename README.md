# FinOps Gatekeeper Provider

FinOps Gatekeeper Provider 是一个为 Open Policy Agent Gatekeeper 提供外部数据的 Webhook 服务。

## 功能特性

- 支持 Mutating 和 Validating Webhook
- 可通过配置文件动态注册多个 Webhook
- 支持动态启用/禁用 Webhook
- 支持自定义 Webhook 路径

## 快速开始

### 构建

```bash
go build -o finops-gatekeeper-provider main.go
```

### 运行

```bash
./finops-gatekeeper-provider
```

## 配置

服务支持通过 YAML 配置文件进行配置。默认情况下，会在当前目录查找 `config.yaml` 文件。

### 配置文件格式

```yaml
server:
  port: 8090           # 服务端口
  timeout: 5           # 请求超时时间(秒)

webhooks:
  - name: "webhook名称"
    path: "/webhook路径"
    type: "webhook类型(mutating或validating)"
    enabled: true      # 是否启用
```

### 默认配置

如果未提供配置文件，将使用以下默认配置：

```yaml
server:
  port: 8090
  timeout: 5

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
```

### Webhook类型映射规则

Webhook处理器根据名称自动映射：
- 名称包含"resources"且类型为"mutating" -> 默认资源配置 Mutator
- 名称包含"labels"且类型为"mutating" -> 默认标签配置 Mutator
- 名称包含"resources"且类型为"validating" -> 资源配置 Validator
- 名称包含"labels"且类型为"validating" -> 标签配置 Validator

## 部署

可以使用 `manifest/` 目录中的 Kubernetes 资源进行部署：

```bash
kubectl apply -f manifest/rbac.yaml
kubectl apply -f manifest/deployment.yaml
kubectl apply -f manifest/service.yaml
```