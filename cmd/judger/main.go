package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/FashOJ/Judger/api/proto/judger"
	"github.com/FashOJ/Judger/internal/discovery"
	"github.com/FashOJ/Judger/internal/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	// 命令行参数
	port := flag.Int("port", 50051, "The server port")
	redisAddr := flag.String("redis", "localhost:6379", "Redis address for service discovery")
	flag.Parse()

	// 初始化 Logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting FashOJ Judger Server...", zap.Int("port", *port))

	// 监听端口
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	// 创建 gRPC 服务器
	s := grpc.NewServer()
	judgeServer := server.NewJudgeServer(logger)
	pb.RegisterJudgeServiceServer(s, judgeServer)

	// 服务注册与发现
	registry := discovery.NewRegistry(*redisAddr, "fashoj-judger", fmt.Sprintf("localhost:%d", *port), logger)
	registry.Start()
	defer registry.Stop()

	// 优雅退出
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		logger.Info("Shutting down server...")
		s.GracefulStop()
		registry.Stop()
	}()

	// 启动服务
	logger.Info("Server listening", zap.String("addr", lis.Addr().String()))
	if err := s.Serve(lis); err != nil {
		logger.Fatal("Failed to serve", zap.Error(err))
	}
}
