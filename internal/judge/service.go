package judge

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/FashOJ/Judger/internal/compiler"
	"github.com/FashOJ/Judger/internal/model"
	"github.com/FashOJ/Judger/internal/runner"
	"github.com/FashOJ/Judger/internal/sandbox"
)

type JudgeService struct {
	runner     runner.Runner
	jobQueue   chan *model.JudgeTask
	cgroupPool *sandbox.CgroupPool
}

func NewJudgeService(workers int, queueSize int) *JudgeService {
	// 初始化 CgroupPool
	// 池大小等于 Worker 数量，因为每个 Worker 同一时间只处理一个任务
	pool, err := sandbox.NewCgroupPool(workers, "fashoj_pool")
	if err != nil {
		// 这里 panic 是合理的，因为如果资源池初始化失败，服务无法运行
		panic(fmt.Sprintf("failed to init cgroup pool: %v", err))
	}

	s := &JudgeService{
		cgroupPool: pool,
		runner:     runner.NewSandboxRunner(pool),
		jobQueue:   make(chan *model.JudgeTask, queueSize),
	}
	s.startWorkers(workers)
	return s
}

func (s *JudgeService) startWorkers(n int) {
	for i := 0; i < n; i++ {
		go s.worker()
	}
}

func (s *JudgeService) worker() {
	for task := range s.jobQueue {
		// 执行判题逻辑
		// 注意：这里的 ctx 暂时使用 Background，或者可以从 task 中传递（如果需要支持取消）
		// 实际生产中，Task 应该包含 Context
		ctx := context.Background()
		result := s.judgeCore(ctx, task)

		// 将结果发送回 ResultChan
		// 使用非阻塞发送防止死锁（虽然理论上接收方在等待）
		select {
		case task.ResultChan <- result:
		default:
			// Log error: result channel blocked or closed
			fmt.Printf("Error: ResultChan blocked for task %s\n", task.ID)
		}
	}
}

// Submit 提交任务
func (s *JudgeService) Submit(task *model.JudgeTask) error {
	select {
	case s.jobQueue <- task:
		return nil
	default:
		return fmt.Errorf("system busy: job queue is full")
	}
}

// judgeCore 核心判题逻辑 (原 Judge 方法)
func (s *JudgeService) judgeCore(ctx context.Context, task *model.JudgeTask) *model.JudgeResult {
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
	exePath, compileLog, err := comp.Compile(ctx, task.SourceCode, task.WorkDir)
	result.CompileLog = compileLog
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
	output, stderr, status, timeUsed, memUsed, err := s.runner.Run(ctx, exePath, inputContent, task.TimeLimit, task.MemoryLimit)

	// 构造基础 CaseResult
	caseRes := model.CaseResult{
		CaseID:     tc.ID,
		Status:     status,
		TimeUsed:   timeUsed,
		MemoryUsed: memUsed,
		Input:      limitString(inputContent, 200), // 限制长度
		Output:     limitString(output, 200),
		// ExpectedOut 暂时为空，下面读取后填充
	}

	// 读取预期输出 (用于对比和返回)
	expectedContent, readErr := getFileContentOrString(tc.ExpectedOut)
	if readErr != nil {
		caseRes.Status = model.StatusSystemError
		caseRes.Message = fmt.Sprintf("failed to read expected output: %v", readErr)
		return caseRes
	}
	caseRes.ExpectedOut = limitString(expectedContent, 200)

	if err != nil {
		caseRes.Message = err.Error()
		if status == model.StatusRuntimeError {
			// 如果是 RE，附带 stderr
			caseRes.Message = fmt.Sprintf("Runtime Error: %s\nStderr: %s", err.Error(), limitString(stderr, 500))
		}
		return caseRes
	}

	if status != model.StatusAccepted {
		caseRes.Status = status
		// 如果是 OLE/MLE 等，Message 可能为空，或者需要补充信息
		if status == model.StatusMemoryLimitExceeded {
			caseRes.Message = "Memory Limit Exceeded"
		}
		return caseRes
	}

	// 比较输出
	if compareOutput(output, expectedContent) {
		caseRes.Status = model.StatusAccepted
		caseRes.Message = "OK"
		return caseRes
	} else if isPresentationError(output, expectedContent) {
		caseRes.Status = model.StatusPresentationError
		caseRes.Message = "Format mismatch"
		return caseRes
	} else {
		caseRes.Status = model.StatusWrongAnswer
		// 生成简短 Diff
		caseRes.Message = generateDiff(expectedContent, output)
		return caseRes
	}
}

// generateDiff 生成简短的对比信息
func generateDiff(expected, actual string) string {
	// 简单截取前 50 个字符展示
	expLimit := limitString(expected, 50)
	actLimit := limitString(actual, 50)
	return fmt.Sprintf("Expected: %q, Got: %q", expLimit, actLimit)
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

// compareOutput 严格比较（包括尾部换行符）
func compareOutput(actual, expected string) bool {
	// 传统的 OJ 通常要求完全一致，或者允许忽略行末空格
	// 这里我们先保留 TrimSpace 的宽容策略作为 AC，如果完全不匹配再检查 PE
	// 但为了支持 PE，我们需要一个更严格的 AC 标准：
	// AC: 内容完全一致（或者仅忽略行末空格）
	// PE: 去除所有空白字符后一致

	// 策略调整：
	// AC: TrimRight 每个行末空格，TrimRight 整个字符串末尾换行
	return strings.TrimSpace(actual) == strings.TrimSpace(expected)
}

// isPresentationError 检查是否为格式错误
func isPresentationError(actual, expected string) bool {
	// 去除所有空白字符（空格、换行、制表符）后比较
	return removeAllWhitespace(actual) == removeAllWhitespace(expected)
}

func removeAllWhitespace(s string) string {
	return strings.Join(strings.Fields(s), "")
}

func limitString(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
