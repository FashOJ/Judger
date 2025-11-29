package judge

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/FashOJ/Judger/internal/compiler"
	"github.com/FashOJ/Judger/internal/model"
	"github.com/FashOJ/Judger/internal/runner"
)

type JudgeService struct {
	runner runner.Runner
}

func NewJudgeService() *JudgeService {
	// 优先尝试使用 SandboxRunner，这通常需要 root 权限
	// 实际生产中可以通过配置决定
	return &JudgeService{
		runner: runner.NewSandboxRunner(),
	}
}

func (s *JudgeService) Judge(ctx context.Context, task *model.JudgeTask) *model.JudgeResult {
	result := &model.JudgeResult{
		Status:      model.StatusAccepted,
		CaseResults: make([]model.CaseResult, 0),
	}

	// 1. 获取编译器
	comp, err := compiler.GetCompiler(task.Language)
	if err != nil {
		result.Status = model.StatusSystemError
		result.Message = err.Error()
		return result
	}

	// 2. 编译
	exePath, err := comp.Compile(task.SourceCode, task.WorkDir)
	if err != nil {
		result.Status = model.StatusCompileError
		result.Message = err.Error()
		return result
	}

	// 3. 运行测试用例
	maxTime := int64(0)
	maxMem := int64(0)

	for _, tc := range task.TestCases {
		caseRes := s.runTestCase(ctx, exePath, task, tc)
		result.CaseResults = append(result.CaseResults, caseRes)

		// 更新最大时间和内存
		if caseRes.TimeUsed > maxTime {
			maxTime = caseRes.TimeUsed
		}
		if caseRes.MemoryUsed > maxMem {
			maxMem = caseRes.MemoryUsed
		}

		// 如果非 AC，则整体状态改变（取第一个非 AC 状态）
		if caseRes.Status != model.StatusAccepted && result.Status == model.StatusAccepted {
			result.Status = caseRes.Status
			result.Message = caseRes.Message
		}
	}

	result.TimeUsed = maxTime
	result.MemoryUsed = maxMem
	return result
}

func (s *JudgeService) runTestCase(ctx context.Context, exePath string, task *model.JudgeTask, tc model.TestCase) model.CaseResult {
	// 读取输入
	inputContent, err := getFileContentOrString(tc.Input)
	if err != nil {
		return model.CaseResult{
			CaseID:  tc.ID,
			Status:  model.StatusSystemError,
			Message: fmt.Sprintf("failed to read input: %v", err),
		}
	}

	// 运行
	output, status, timeUsed, memUsed, err := s.runner.Run(ctx, exePath, inputContent, task.TimeLimit, task.MemoryLimit)
	if err != nil {
		return model.CaseResult{
			CaseID:   tc.ID,
			Status:   status,
			TimeUsed: timeUsed,
			Message:  err.Error(),
		}
	}

	if status != model.StatusAccepted {
		return model.CaseResult{
			CaseID:     tc.ID,
			Status:     status,
			TimeUsed:   timeUsed,
			MemoryUsed: memUsed,
		}
	}

	// 比较输出
	expectedContent, err := getFileContentOrString(tc.ExpectedOut)
	if err != nil {
		return model.CaseResult{
			CaseID:  tc.ID,
			Status:  model.StatusSystemError,
			Message: fmt.Sprintf("failed to read expected output: %v", err),
		}
	}

	if compareOutput(output, expectedContent) {
		return model.CaseResult{
			CaseID:     tc.ID,
			Status:     model.StatusAccepted,
			TimeUsed:   timeUsed,
			MemoryUsed: memUsed,
			Message:    "OK",
		}
	} else {
		return model.CaseResult{
			CaseID:     tc.ID,
			Status:     model.StatusWrongAnswer,
			TimeUsed:   timeUsed,
			MemoryUsed: memUsed,
			Message:    fmt.Sprintf("Expected: %s, Got: %s", limitString(expectedContent, 50), limitString(output, 50)),
		}
	}
}

// getFileContentOrString 判断是文件路径还是内容，如果是文件则读取
func getFileContentOrString(input string) (string, error) {
	// 简单判断：如果文件存在则读取，否则视为内容
	// 注意：这种判断有歧义，实际生产中最好明确区分
	if _, err := os.Stat(input); err == nil {
		content, err := os.ReadFile(input)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}
	return input, nil
}

// compareOutput 简单的文本比较
func compareOutput(actual, expected string) bool {
	return strings.TrimSpace(actual) == strings.TrimSpace(expected)
}

func limitString(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
