# 使用官方Go镜像作为构建环境
FROM golang:1.24 AS builder

# 设置工作目录
WORKDIR /app

# 复制源代码和vendor目录
COPY . .

# 构建应用（使用vendor目录，禁用CGO）
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o finops-gatekeeper-provider .

# 使用轻量级镜像作为运行环境
FROM alpine:latest

# 安装ca证书以支持HTTPS请求
RUN apk --no-cache add ca-certificates

# 创建非root用户
RUN adduser -D -s /bin/sh appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/finops-gatekeeper-provider .

# 复制配置文件
COPY --from=builder /app/config.yaml .

# 更改文件所有者
RUN chown -R appuser:appuser /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8090

# 运行应用
ENTRYPOINT ["./finops-gatekeeper-provider"]