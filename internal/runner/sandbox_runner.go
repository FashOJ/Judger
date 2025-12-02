package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/FashOJ/Judger/internal/model"
	"github.com/FashOJ/Judger/internal/sandbox"
)

const (
	// MaxOutputSize 最大输出限制 (16MB)
	MaxOutputSize = 16 * 1024 * 1024
)

type SandboxRunner struct {
	CgroupPool *sandbox.CgroupPool
}

func NewSandboxRunner(cgroupPool *sandbox.CgroupPool) *SandboxRunner {
	return &SandboxRunner{
		CgroupPool: cgroupPool,
	}
}

func (r *SandboxRunner) Run(ctx context.Context, exePath string, input string, timeLimit int64, memoryLimit int64) (string, string, model.JudgeStatus, int64, int64, error) {
	// 1. 准备临时文件用于 IO 重定向
	tmpDir := filepath.Dir(exePath)
	inputFile := filepath.Join(tmpDir, "input.temp")
	outputFile := filepath.Join(tmpDir, "output.temp")
	errorFile := filepath.Join(tmpDir, "error.temp")

	if err := os.WriteFile(inputFile, []byte(input), 0644); err != nil {
		return "", "", model.StatusSystemError, 0, 0, fmt.Errorf("write input failed: %v", err)
	}

	// 2. 获取 Cgroup (从池中)
	cgroup := r.CgroupPool.Acquire()
	defer r.CgroupPool.Release(cgroup)

	// 设置资源限制 (内存 + 10% buffer, CPU 100%)
	memLimitBytes := (memoryLimit * 1024 * 1024)
	// 给一点 buffer 防止瞬间 OOM，实际判断以 usage 为准，或者依靠 OOM Kill
	_ = cgroup.SetMemoryLimit(memLimitBytes + 1024*1024)
	_ = cgroup.SetCPULimit(100)

	// 3. 准备沙箱命令
	cmd, err := sandbox.RunInSandbox(exePath, []string{}, "", inputFile, outputFile, errorFile)
	if err != nil {
		return "", "", model.StatusSystemError, 0, 0, fmt.Errorf("prepare sandbox failed: %v", err)
	}

	// 4. 启动进程
	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return "", "", model.StatusRuntimeError, 0, 0, fmt.Errorf("start process failed: %v", err)
	}

	// 将进程加入 Cgroup
	if err := cgroup.AddProcess(cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		return "", "", model.StatusSystemError, 0, 0, fmt.Errorf("add process to cgroup failed: %v", err)
	}

	// 5. 等待结束或超时
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var status model.JudgeStatus = model.StatusAccepted
	var timeUsed int64
	var memoryUsed int64
	var runErr error // 用于捕获 <-done 的错误

	timeoutDuration := time.Duration(timeLimit) * time.Millisecond
	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		status = model.StatusSystemError
	case <-time.After(timeoutDuration):
		_ = cmd.Process.Kill()
		status = model.StatusTimeLimitExceeded
		timeUsed = timeLimit
	case runErr = <-done:
		// 正常结束
	}

	// 获取 CPU 使用 (从 cgroup)
	if cpuTime, err := cgroup.GetCPUUsage(); err == nil {
		// 如果 cgroup 能获取到准确的 CPU 时间，则优先使用
		timeUsed = cpuTime
	} else {
		// 否则使用 wall time
		// 注意：如果是超时被 kill，这里的 wall time 可能不准确，timeUsed 已经在上面赋值为 limit
		if status != model.StatusTimeLimitExceeded {
			timeUsed = time.Since(startTime).Milliseconds()
		}
	}

	// 检查运行错误 (仅当非超时状态时)
	if status == model.StatusAccepted && runErr != nil {
		// 检查退出码和信号
		if exitError, ok := runErr.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)

			// 1. 检查是否被信号杀死
			if ws.Signaled() {
				sig := ws.Signal()
				switch sig {
				case syscall.SIGKILL:
					// 可能是 OOM，也可能是 TLE (但在 time.After 分支处理 TLE)
					// 结合内存使用判断 MLE
					status = model.StatusRuntimeError // 暂定，下面会修正
				case syscall.SIGSEGV:
					status = model.StatusRuntimeError // Segmentation Fault
				default:
					status = model.StatusRuntimeError
				}
			} else {
				// 非 0 退出码
				status = model.StatusRuntimeError
			}
		} else {
			status = model.StatusRuntimeError
		}
	}

	// 获取内存使用 (从 cgroup)
	if mem, err := cgroup.GetMemoryUsage(); err == nil {
		memoryUsed = mem / 1024 // Convert to KB
	}

	// 检查 MLE (Memory Limit Exceeded)
	// 如果内存使用超过限制，或者因为 SIGKILL 退出且内存很高
	if memoryUsed > memoryLimit*1024 {
		status = model.StatusMemoryLimitExceeded
	} else if status == model.StatusRuntimeError {
		// 如果是被 KILL 且接近内存限制，大概率是 OOM
		// 这是一个启发式判断，不完全准确
		if memoryUsed > int64(float64(memoryLimit*1024)*0.9) {
			status = model.StatusMemoryLimitExceeded
		}
	}

	// 读取输出
	outputBytes, _ := os.ReadFile(outputFile)
	if len(outputBytes) > MaxOutputSize {
		outputBytes = outputBytes[:MaxOutputSize]
	}

	// 读取 stderr
	errorBytes, _ := os.ReadFile(errorFile)

	// 清理临时文件
	_ = os.Remove(inputFile)
	_ = os.Remove(outputFile)
	_ = os.Remove(errorFile)

	return string(outputBytes), string(errorBytes), status, timeUsed, memoryUsed, nil
}
