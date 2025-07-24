# FinOps Gatekeeper Provider

一个支持Mutating和Admission两种Webhook模式的Gatekeeper External Data Provider模板项目。

## 项目结构

```
.
├── main.go                 # 程序入口
├── go.mod                  # Go模块定义
├── go.sum                  # Go依赖校验和
├── pkg/
│   └── webhook/
│       └── server.go       # Webhook服务器实现
└── manifest/
    ├── deployment.yaml     # Kubernetes部署文件
    ├── rbac.yaml           # RBAC权限配置
    └── service.yaml        # 服务配置
```

## 功能特性

- 支持Mutating和Validating两种Webhook模式
- 可配置的路由路径和超时设置
- 结构化的日志记录
- 优雅的服务启停
- 支持Gatekeeper External Data API

## 快速开始

### 构建

```bash
go build -o finops-gatekeeper-provider .
```

### 运行

```bash
./finops-gatekeeper-provider
```

或者直接运行:

```bash
go run main.go
```

### 配置

可以通过修改[main.go](file:///Users/ringtail/go/src/github.com/AliyunContainerService/finops-gatekeeper-provider/main.go)中的配置来调整服务参数:

```go
config := webhook.Config{
    Port:         8090,        // 服务端口
    MutatePath:   "/mutate",   // Mutating webhook路径
    ValidatePath: "/validate", // Validating webhook路径
    Timeout:      5 * time.Second, // 请求超时时间
    Logger:       log,         // 日志记录器
}
```

## 自定义业务逻辑

在[pkg/webhook/server.go](file:///Users/ringtail/go/src/github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/webhook/server.go)中可以找到两个主要的处理函数:

1. `Mutate` - 处理mutating webhook请求
2. `Validate` - 处理validating webhook请求

根据你的业务需求修改这两个函数的实现。

## 部署到Kubernetes

1. 构建Docker镜像
2. 修改[manifest/deployment.yaml](file:///Users/ringtail/go/src/github.com/AliyunContainerService/finops-gatekeeper-provider/manifest/deployment.yaml)中的镜像地址
3. 应用配置:

```bash
kubectl apply -f manifest/rbac.yaml
kubectl apply -f manifest/deployment.yaml
kubectl apply -f manifest/service.yaml
```