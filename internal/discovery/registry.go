package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Registry struct {
	client      *redis.Client
	serviceName string
	instanceID  string
	addr        string
	logger      *zap.Logger
	stopChan    chan struct{}
	once        sync.Once
}

type InstanceInfo struct {
	ID          string  `json:"id"`
	Addr        string  `json:"addr"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	TaskCount   int     `json:"task_count"`
	LastUpdated int64   `json:"last_updated"`
}

func NewRegistry(redisAddr, serviceName, addr string, logger *zap.Logger) *Registry {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	hostname, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s-%s", hostname, addr)

	return &Registry{
		client:      rdb,
		serviceName: serviceName,
		instanceID:  instanceID,
		addr:        addr,
		logger:      logger,
		stopChan:    make(chan struct{}),
	}
}

func (r *Registry) Start() {
	go r.heartbeat()
}

func (r *Registry) Stop() {
	r.once.Do(func() {
		close(r.stopChan)
		// 删除注册信息
		key := fmt.Sprintf("judger:instances:%s", r.instanceID)
		r.client.Del(context.Background(), key)
	})
}

func (r *Registry) heartbeat() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.register()
		}
	}
}

func (r *Registry) register() {
	// 这里可以接入 gopsutil 获取真实的系统负载
	// 暂时使用模拟数据
	info := InstanceInfo{
		ID:          r.instanceID,
		Addr:        r.addr,
		CPUUsage:    0.0, // TODO: Get real CPU usage
		MemoryUsage: 0.0, // TODO: Get real Memory usage
		TaskCount:   0,   // TODO: Get real task count
		LastUpdated: time.Now().Unix(),
	}

	data, _ := json.Marshal(info)
	key := fmt.Sprintf("judger:instances:%s", r.instanceID)

	// 设置 15 秒过期，心跳间隔 5 秒
	err := r.client.Set(context.Background(), key, data, 15*time.Second).Err()
	if err != nil {
		r.logger.Error("Failed to send heartbeat", zap.Error(err))
	} else {
		// r.logger.Debug("Heartbeat sent", zap.String("instance_id", r.instanceID))
	}
}
