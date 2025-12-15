package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/FashOJ/Judger/internal/config"
	"github.com/FashOJ/Judger/internal/model"
	"github.com/FashOJ/Judger/internal/sandbox"
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

	// 确保临时文件权限允许 nobody 用户读取/写入
	// input: 644 (owner write, others read) -> nobody (others) can read. OK.
	// output/error: will be created by nobody?
	// No, they are created by parent (root) via os.Create?
	// Wait, os.Create in namespaces.go happens BEFORE switching user.
	// If parent creates them, they are owned by root.
	// We need to ensure nobody can write to them.
	// Change mode to 0666 or chown.
	_ = os.Chmod(inputFile, 0666)
	_ = os.Chmod(outputFile, 0666) // Pre-create if needed or rely on sandbox logic
	_ = os.Chmod(errorFile, 0666)

	// Better: Pre-create output/error files and chmod them
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		os.WriteFile(outputFile, []byte(""), 0666)
	}
	_ = os.Chmod(outputFile, 0666)

	if _, err := os.Stat(errorFile); os.IsNotExist(err) {
		os.WriteFile(errorFile, []byte(""), 0666)
	}
	_ = os.Chmod(errorFile, 0666)

	// 2. 获取 Cgroup (从池中)
	cgroup := r.CgroupPool.Acquire()
	defer r.CgroupPool.Release(cgroup)

	// 设置资源限制 (内存 + 10% buffer, CPU 100%)
	memLimitBytes := (memoryLimit * 1024 * 1024)
	// 给一点 buffer 防止瞬间 OOM，实际判断以 usage 为准，或者依靠 OOM Kill
	_ = cgroup.SetMemoryLimit(memLimitBytes + 1024*1024)
	_ = cgroup.SetCPULimit(100)

	// 3. 准备沙箱命令
	// 注意：config.GlobalConfig.Sandbox.CgroupRoot 只是 Cgroup 的名字，不是文件系统路径。
	// 目前 RunInSandbox 只有在 rootFS 非空时才会检查路径存在。
	// 为了避免报错，我们暂时传空字符串，因为我们还没有构建真正的 rootfs。
	// 如果未来要支持 chroot，需要传入真实的 rootfs 路径（例如 /var/lib/fashoj/rootfs/base）。
	cmd, err := sandbox.RunInSandbox(exePath, []string{}, "", inputFile, outputFile, errorFile)
	if err != nil {
		return "", "", model.StatusSystemError, 0, 0, fmt.Errorf("prepare sandbox failed: %v", err)
	}

	// 4. 启动进程
	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return "", "", model.StatusRuntimeError, 0, 0, fmt.Errorf("start process failed: %v", err)
	}

	// 设置栈空间限制 (RLIMIT_STACK)
	// 默认设置为内存限制的大小，或者给一个较大的固定值 (如 128MB)
	// C++ 程序经常需要较大栈空间
	stackLimitBytes := memLimitBytes
	_ = sandbox.SetStackLimit(cmd.Process.Pid, stackLimitBytes)

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
	if int64(len(outputBytes)) > config.GlobalConfig.Sandbox.MaxOutputSize {
		outputBytes = outputBytes[:config.GlobalConfig.Sandbox.MaxOutputSize]
	}

	// 读取 stderr
	errorBytes, _ := os.ReadFile(errorFile)

	// 清理临时文件
	_ = os.Remove(inputFile)
	_ = os.Remove(outputFile)
	_ = os.Remove(errorFile)

	return string(outputBytes), string(errorBytes), status, timeUsed, memoryUsed, nil
}
