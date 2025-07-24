package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AliyunContainerService/finops-gatekeeper-provider/pkg/webhook"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

var log logr.Logger

func main() {
	// 初始化日志
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("unable to initialize logger: %v", err))
	}
	log = zapr.NewLogger(zapLog)
	log.WithName("finops-gatekeeper-provider")

	// 创建webhook服务器配置
	config := webhook.Config{
		Port:         8090,
		MutatePath:   "/mutate",
		ValidatePath: "/validate",
		Timeout:      5 * time.Second,
		Logger:       log,
	}

	// 创建webhook服务器
	server := webhook.NewServer(config)

	// 启动服务器
	go func() {
		log.Info("starting server on port 8090")
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Error(err, "server failed to start")
			panic(err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down server...")

	// 优雅地关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Error(err, "server forced to shutdown")
	}

	log.Info("server exited")
}
