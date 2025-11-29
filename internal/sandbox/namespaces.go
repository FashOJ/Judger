package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// RunInSandbox 在隔离环境中运行命令
// 注意：这需要在 Linux 环境下运行，且需要 root 权限
func RunInSandbox(cmdPath string, args []string, rootFS string, inputPath, outputPath, errorPath string) (*exec.Cmd, error) {
	cmd := exec.Command(cmdPath, args...)

	// 设置 Namespace 隔离
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | // 主机名隔离
			syscall.CLONE_NEWPID | // PID 隔离
			syscall.CLONE_NEWNS | // 挂载点隔离
			syscall.CLONE_NEWNET | // 网络隔离 (无网络)
			syscall.CLONE_NEWIPC, // IPC 隔离
		// 不使用 CLONE_NEWUSER，因为它会导致文件权限问题，除非做 uid mapping
	}

	// 设置 Chroot (如果提供了 rootFS)
	// 注意：如果使用 Chroot，cmdPath 必须是相对于 rootFS 的路径，或者在 rootFS 内可访问
	// 这里为了简化，假设外部已经准备好了环境，或者暂时不启用严格的 chroot，只做 namespace 隔离
	// 如果要严格 Chroot，需要在 cmd.PreStart 钩子中执行 syscall.Chroot 和 syscall.Chdir
	if rootFS != "" {
		// 确保 rootFS 存在
		if _, err := os.Stat(rootFS); os.IsNotExist(err) {
			return nil, fmt.Errorf("rootFS does not exist: %s", rootFS)
		}
		// 简单的 Chroot 设置 (实际生产需要更复杂的 mount 绑定)
		// cmd.Dir = "/"
		// 这里我们暂时不直接使用 cmd.SysProcAttr.Chroot，因为它可能需要手动 mount /proc 等
	}

	// 设置输入输出重定向
	if inputPath != "" {
		fin, err := os.Open(inputPath)
		if err != nil {
			return nil, err
		}
		cmd.Stdin = fin
	}

	if outputPath != "" {
		fout, err := os.Create(outputPath)
		if err != nil {
			return nil, err
		}
		cmd.Stdout = fout
	}

	if errorPath != "" {
		ferr, err := os.Create(errorPath)
		if err != nil {
			return nil, err
		}
		cmd.Stderr = ferr
	}

	// 设置环境变量
	cmd.Env = []string{"PATH=/bin:/usr/bin", "HOME=/"}

	return cmd, nil
}
