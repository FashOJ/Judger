package sandbox

import (
	"fmt"
	"sync"
)

// CgroupPool Cgroup 池
type CgroupPool struct {
	mu       sync.Mutex
	pool     chan *CgroupManager
	size     int
	basePath string
	prefix   string
}

// NewCgroupPool 创建 Cgroup 池
func NewCgroupPool(size int, prefix string) (*CgroupPool, error) {
	pool := &CgroupPool{
		pool:     make(chan *CgroupManager, size),
		size:     size,
		prefix:   prefix,
		basePath: "fashoj_judger", // 默认根路径
	}

	// 预先创建 Cgroup
	for i := 0; i < size; i++ {
		name := fmt.Sprintf("%s_%d", prefix, i)
		cg, err := NewCgroupManager(name)
		if err != nil {
			return nil, fmt.Errorf("failed to init cgroup %s: %v", name, err)
		}
		pool.pool <- cg
	}

	return pool, nil
}

// Acquire 获取一个 Cgroup
func (p *CgroupPool) Acquire() *CgroupManager {
	return <-p.pool
}

// Release 归还并重置 Cgroup
func (p *CgroupPool) Release(cg *CgroupManager) {
	// 清理进程（确保 cgroup.procs 为空）
	// 注意：在 Cgroup v2 中，只要进程退出了，这里通常是空的。
	// 如果有残留进程（僵尸进程），需要 kill 掉。
	// 这里简单重置一下参数（可选，因为下次 Acquire 会重新 Set）
	
	p.pool <- cg
}

// Destroy 销毁池
func (p *CgroupPool) Destroy() {
	close(p.pool)
	for cg := range p.pool {
		_ = cg.Destroy()
	}
}
