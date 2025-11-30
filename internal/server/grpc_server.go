package server

import (
	"context"
	"fmt"
	"os"

	pb "github.com/FashOJ/Judger/api/proto/judger"
	"github.com/FashOJ/Judger/internal/judge"
	"github.com/FashOJ/Judger/internal/model"
	"go.uber.org/zap"
)

type JudgeServer struct {
	pb.UnimplementedJudgeServiceServer
	judgeService *judge.JudgeService
	logger       *zap.Logger
}

func NewJudgeServer(workers int, queueSize int, logger *zap.Logger) *JudgeServer {
	return &JudgeServer{
		judgeService: judge.NewJudgeService(workers, queueSize),
		logger:       logger,
	}
}

func (s *JudgeServer) Judge(ctx context.Context, req *pb.JudgeRequest) (*pb.JudgeResponse, error) {
	s.logger.Info("Received judge task", zap.String("task_id", req.Id), zap.String("language", req.Language))

	// 1. 转换请求模型
	workDir := req.WorkDir
	if workDir == "" {
		workDir = fmt.Sprintf("temp/%s", req.Id)
		_ = os.MkdirAll(workDir, 0755)
		// 注意：如果是异步处理，这里的 defer os.RemoveAll 会在 Submit 后立即执行，导致工作目录被删
		// 所以不能在这里 defer，必须在 Worker 处理完后清理，或者由 Worker 清理
		// 为了简单，我们暂且保留同步等待逻辑，所以 defer 依然有效，
		// 但如果是真正的异步投递后立即返回 ID，则需要 Worker 清理。
		// 本次重构 gRPC 依然是同步接口 (Req -> Resp)，只是内部排队。
		// 所以我们会在 ResultChan 返回后清理。
	}
	defer os.RemoveAll(workDir) // 清理临时目录

	task := &model.JudgeTask{
		ID:          req.Id,
		SourceCode:  req.SourceCode,
		Language:    model.Language(req.Language),
		TimeLimit:   req.TimeLimit,
		MemoryLimit: req.MemoryLimit,
		WorkDir:     workDir,
		TestCases:   make([]model.TestCase, len(req.TestCases)),
		ResultChan:  make(chan *model.JudgeResult, 1), // 初始化结果通道
	}

	for i, tc := range req.TestCases {
		task.TestCases[i] = model.TestCase{
			ID:          tc.Id,
			Input:       tc.Input,
			ExpectedOut: tc.ExpectedOutput,
		}
	}

	// 2. 提交任务到队列
	if err := s.judgeService.Submit(task); err != nil {
		s.logger.Warn("Judge queue full", zap.String("task_id", req.Id))
		return nil, err // 返回 gRPC 错误
	}

	// 3. 等待结果 (同步转异步再转同步)
	var result *model.JudgeResult
	select {
	case result = <-task.ResultChan:
		// 收到结果
	case <-ctx.Done():
		// 客户端取消或超时
		s.logger.Warn("Client cancelled or timed out", zap.String("task_id", req.Id))
		return nil, ctx.Err()
	}

	// 4. 转换响应模型
	resp := &pb.JudgeResponse{
		Status:      string(result.Status),
		Message:     result.Message,
		TimeUsed:    result.TimeUsed,
		MemoryUsed:  result.MemoryUsed,
		CompileLog:  result.CompileLog,
		CaseResults: make([]*pb.CaseResult, len(result.CaseResults)),
	}

	for i, cr := range result.CaseResults {
		resp.CaseResults[i] = &pb.CaseResult{
			CaseId:         cr.CaseID,
			Status:         string(cr.Status),
			TimeUsed:       cr.TimeUsed,
			MemoryUsed:     cr.MemoryUsed,
			Message:        cr.Message,
			Input:          cr.Input,
			Output:         cr.Output,
			ExpectedOutput: cr.ExpectedOut,
		}
	}

	s.logger.Info("Judge task completed", zap.String("task_id", req.Id), zap.String("status", resp.Status))
	return resp, nil
}
