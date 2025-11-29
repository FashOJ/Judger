package runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/FashOJ/Judger/internal/model"
)

type Runner interface {
	Run(ctx context.Context, exePath string, input string, timeLimit int64, memoryLimit int64) (string, model.JudgeStatus, int64, int64, error)
}

type LocalRunner struct{}

func NewLocalRunner() *LocalRunner {
	return &LocalRunner{}
}

func (r *LocalRunner) Run(ctx context.Context, exePath string, input string, timeLimit int64, memoryLimit int64) (string, model.JudgeStatus, int64, int64, error) {
	// 创建带超时的上下文
	timeoutDuration := time.Duration(timeLimit) * time.Millisecond
	// 稍微多给一点缓冲时间，用于系统开销
	ctx, cancel := context.WithTimeout(ctx, timeoutDuration+100*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, exePath)

	// 设置输入
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime).Milliseconds()

	// 检查超时
	if ctx.Err() == context.DeadlineExceeded {
		return "", model.StatusTimeLimitExceeded, duration, 0, nil
	}

	if err != nil {
		// 检查是否是信号中断（如内存超限被kill，这里本地运行很难精确捕捉OOM，暂时简单处理）
		if exitError, ok := err.(*exec.ExitError); ok {
			// 这里可以根据信号判断，例如 SIGKILL 可能是 OOM
			_ = exitError
			return "", model.StatusRuntimeError, duration, 0, fmt.Errorf("runtime error: %s", stderr.String())
		}
		return "", model.StatusRuntimeError, duration, 0, err
	}

	// 简单的内存检查（本地运行很难精确获取峰值内存，暂时返回0或简单估算）
	// 实际生产中需要配合 cgroups 或 getrusage
	memoryUsed := int64(0)

	// 检查实际运行时间是否超过题目限制（CommandContext 的超时是硬性终止，这里是逻辑判断）
	if duration > timeLimit {
		return "", model.StatusTimeLimitExceeded, duration, memoryUsed, nil
	}

	return stdout.String(), model.StatusAccepted, duration, memoryUsed, nil
}
