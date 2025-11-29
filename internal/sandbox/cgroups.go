package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CgroupManager 管理 Cgroup V2
type CgroupManager struct {
	RootPath string
	Name     string
}

func NewCgroupManager(name string) (*CgroupManager, error) {
	// 默认 cgroup v2 挂载点
	const cgroupRoot = "/sys/fs/cgroup"
	path := filepath.Join(cgroupRoot, name)

	// 检查是否已存在，如果存在则清理
	if _, err := os.Stat(path); err == nil {
		// 尝试删除旧的 cgroup (注意：如果有进程在运行，删除会失败)
		_ = os.Remove(path)
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cgroup directory: %v", err)
	}

	// 启用控制器
	// 在根 cgroup 启用 cpu 和 memory (通常需要 root 权限操作 cgroup.subtree_control)
	// 注意：在容器内或非特权环境可能受限

	return &CgroupManager{
		RootPath: path,
		Name:     name,
	}, nil
}

// SetMemoryLimit 设置内存限制 (bytes)
func (c *CgroupManager) SetMemoryLimit(limitBytes int64) error {
	limitPath := filepath.Join(c.RootPath, "memory.max")
	// 写入内存限制
	return os.WriteFile(limitPath, []byte(fmt.Sprintf("%d", limitBytes)), 0644)
}

// SetCPULimit 设置 CPU 限制 (quota/period)
// cpuPercent: 100 表示 1 核
func (c *CgroupManager) SetCPULimit(cpuPercent int) error {
	// 默认 period 100000us (100ms)
	period := 100000
	quota := period * cpuPercent / 100

	maxPath := filepath.Join(c.RootPath, "cpu.max")
	return os.WriteFile(maxPath, []byte(fmt.Sprintf("%d %d", quota, period)), 0644)
}

// AddProcess 将进程加入 cgroup
func (c *CgroupManager) AddProcess(pid int) error {
	procsPath := filepath.Join(c.RootPath, "cgroup.procs")
	return os.WriteFile(procsPath, []byte(fmt.Sprintf("%d", pid)), 0644)
}

// GetMemoryUsage 获取当前内存使用量
func (c *CgroupManager) GetMemoryUsage() (int64, error) {
	usagePath := filepath.Join(c.RootPath, "memory.current")
	content, err := os.ReadFile(usagePath)
	if err != nil {
		return 0, err
	}
	var usage int64
	fmt.Sscanf(strings.TrimSpace(string(content)), "%d", &usage)
	return usage, nil
}

// Destroy 清理 cgroup
func (c *CgroupManager) Destroy() error {
	// 需要先移除所有进程，或者 kill 掉
	// 这里简单尝试删除目录
	return os.Remove(c.RootPath)
}
