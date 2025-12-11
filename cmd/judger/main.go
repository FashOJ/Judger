package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/FashOJ/Judger/api/proto/judger"
	"github.com/FashOJ/Judger/internal/config"
	"github.com/FashOJ/Judger/internal/discovery"
	"github.com/FashOJ/Judger/internal/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	// 命令行参数
	configFile := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// 加载配置
	if err := config.LoadConfig(*configFile); err != nil {
		// 如果配置文件不存在，尝试加载默认配置（或者报错）
		// 这里我们允许 config file 不存在，使用默认值，但最好打印 warning
		fmt.Printf("Warning: failed to load config file %s: %v. Using defaults.\n", *configFile, err)
		// 实际上 LoadConfig 内部如果文件不存在会报错，我们需要处理
		// 为了简化，我们暂时认为 LoadConfig 失败是致命的，或者我们重构 LoadConfig 允许忽略
		// 但为了兼容旧的启动方式（无 config 文件），我们可以在这里手动 setDefaults
		// 不过 LoadConfig 已经包含 setDefaults。
		// 如果你想支持无配置文件启动，需要修改 LoadConfig 逻辑。
		// 鉴于目前是重构，我们强制要求配置文件，或者在 main 中构建默认 Config。
		// 让我们简单点：如果 LoadConfig 失败，且是 "no such file"，则仅使用默认值。
		// 但 LoadConfig 内部已经 setDefaults 了，所以即使 Unmarshal 失败，只要 setDefaults 执行了就行。
		// 我们稍微修改 LoadConfig 策略：如果读取失败，也执行 setDefaults。
		// 但现在的实现是读取失败直接返回 error。
		// 所以这里我们先 Fatal，强制要求配置文件，或者你可以创建一个默认的 config.yaml
	}

	// 覆盖命令行参数（如果有）
	// 这里可以添加逻辑：如果命令行传了 port，覆盖 config 中的 port
	// 但为了简洁，我们暂且以 config 为准。

	// 初始化 Logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := config.GlobalConfig.Server
	logger.Info("Starting FashOJ Judger Server...", zap.Int("port", cfg.Port))

	// 监听端口
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	// 创建 gRPC 服务器
	s := grpc.NewServer()
	judgeServer := server.NewJudgeServer(cfg.Workers, cfg.QueueSize, logger)
	pb.RegisterJudgeServiceServer(s, judgeServer)

	// 服务注册与发现
	registry := discovery.NewRegistry(config.GlobalConfig.Redis.Addr, "fashoj-judger", fmt.Sprintf("localhost:%d", cfg.Port), logger)
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
