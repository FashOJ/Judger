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

func NewJudgeServer(logger *zap.Logger) *JudgeServer {
	return &JudgeServer{
		judgeService: judge.NewJudgeService(),
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
		defer os.RemoveAll(workDir) // 清理临时目录
	}

	task := &model.JudgeTask{
		ID:          req.Id,
		SourceCode:  req.SourceCode,
		Language:    model.Language(req.Language),
		TimeLimit:   req.TimeLimit,
		MemoryLimit: req.MemoryLimit,
		WorkDir:     workDir,
		TestCases:   make([]model.TestCase, len(req.TestCases)),
	}

	for i, tc := range req.TestCases {
		task.TestCases[i] = model.TestCase{
			ID:          tc.Id,
			Input:       tc.Input,
			ExpectedOut: tc.ExpectedOutput,
		}
	}

	// 2. 调用核心判题逻辑
	result := s.judgeService.Judge(ctx, task)

	// 3. 转换响应模型
	resp := &pb.JudgeResponse{
		Status:     string(result.Status),
		Message:    result.Message,
		TimeUsed:   result.TimeUsed,
		MemoryUsed: result.MemoryUsed,
		CompileLog: result.CompileLog,
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
