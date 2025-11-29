package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/FashOJ/Judger/internal/model"
	"github.com/FashOJ/Judger/internal/sandbox"
)

type SandboxRunner struct {
	CgroupRoot string
}

func NewSandboxRunner() *SandboxRunner {
	return &SandboxRunner{
		CgroupRoot: "fashoj_judger",
	}
}

func (r *SandboxRunner) Run(ctx context.Context, exePath string, input string, timeLimit int64, memoryLimit int64) (string, model.JudgeStatus, int64, int64, error) {
	// 1. 准备临时文件用于 IO 重定向
	tmpDir := filepath.Dir(exePath)
	inputFile := filepath.Join(tmpDir, "input.temp")
	outputFile := filepath.Join(tmpDir, "output.temp")
	errorFile := filepath.Join(tmpDir, "error.temp")

	if err := os.WriteFile(inputFile, []byte(input), 0644); err != nil {
		return "", model.StatusSystemError, 0, 0, fmt.Errorf("write input failed: %v", err)
	}

	// 2. 创建 Cgroup
	cgName := fmt.Sprintf("%s_%d", r.CgroupRoot, time.Now().UnixNano())
	cgroup, err := sandbox.NewCgroupManager(cgName)
	if err != nil {
		// 如果创建 cgroup 失败（可能是权限不足），降级处理或报错
		// 这里为了演示，如果失败则记录日志并返回系统错误
		return "", model.StatusSystemError, 0, 0, fmt.Errorf("create cgroup failed: %v (try running as root)", err)
	}
	defer cgroup.Destroy()

	// 设置资源限制 (内存 + 10% buffer, CPU 100%)
	memLimitBytes := (memoryLimit * 1024 * 1024)
	_ = cgroup.SetMemoryLimit(memLimitBytes)
	_ = cgroup.SetCPULimit(100)

	// 3. 准备沙箱命令
	cmd, err := sandbox.RunInSandbox(exePath, []string{}, "", inputFile, outputFile, errorFile)
	if err != nil {
		return "", model.StatusSystemError, 0, 0, fmt.Errorf("prepare sandbox failed: %v", err)
	}

	// 4. 启动进程
	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return "", model.StatusRuntimeError, 0, 0, fmt.Errorf("start process failed: %v", err)
	}

	// 将进程加入 Cgroup
	if err := cgroup.AddProcess(cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		return "", model.StatusSystemError, 0, 0, fmt.Errorf("add process to cgroup failed: %v", err)
	}

	// 5. 等待结束或超时
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var status model.JudgeStatus = model.StatusAccepted
	var timeUsed int64
	var memoryUsed int64

	timeoutDuration := time.Duration(timeLimit) * time.Millisecond
	select {
	case <-ctx.Done():
		// 上下文取消
		_ = cmd.Process.Kill()
		status = model.StatusSystemError
	case <-time.After(timeoutDuration):
		// 超时
		_ = cmd.Process.Kill()
		status = model.StatusTimeLimitExceeded
		timeUsed = timeLimit
	case err := <-done:
		// 正常结束或运行时错误
		timeUsed = time.Since(startTime).Milliseconds()
		if err != nil {
			status = model.StatusRuntimeError
		}
	}

	// 获取内存使用 (从 cgroup)
	// 注意：进程结束后可能无法获取 current，最好有一个监控协程，或者取 max_usage
	// 这里简单尝试读取
	if mem, err := cgroup.GetMemoryUsage(); err == nil {
		memoryUsed = mem / 1024 // Convert to KB
	}

	// 检查内存超限 (Cgroup OOM 会导致进程被 Kill)
	if memoryUsed > memoryLimit*1024 { // memoryLimit is MB, memoryUsed is KB
		status = model.StatusMemoryLimitExceeded
	}

	// 读取输出
	outputBytes, _ := os.ReadFile(outputFile)
	
	// 清理临时文件
	_ = os.Remove(inputFile)
	_ = os.Remove(outputFile)
	_ = os.Remove(errorFile)

	return string(outputBytes), status, timeUsed, memoryUsed, nil
}
