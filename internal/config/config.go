package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// GlobalConfig 全局配置单例
var GlobalConfig Config

// Config 总配置结构
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Redis     RedisConfig     `yaml:"redis"`
	Sandbox   SandboxConfig   `yaml:"sandbox"`
	Compilers CompilerConfig  `yaml:"compilers"`
}

// ServerConfig 服务端配置
type ServerConfig struct {
	Port      int `yaml:"port"`
	Workers   int `yaml:"workers"`
	QueueSize int `yaml:"queue_size"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// SandboxConfig 沙箱配置
type SandboxConfig struct {
	CgroupRoot    string `yaml:"cgroup_root"`
	PoolSize      int    `yaml:"pool_size"`
	MaxOutputSize int64  `yaml:"max_output_size"` // bytes
}

// CompilerConfig 编译器/解释器路径配置
type CompilerConfig struct {
	CPP    string `yaml:"cpp"`    // g++ path
	Python string `yaml:"python"` // python interpreter path
	Java   string `yaml:"java"`   // javac path
}

// LoadConfig 加载配置文件
func LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	if err := yaml.Unmarshal(data, &GlobalConfig); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	// 设置默认值
	setDefaults()
	
	return nil
}

func setDefaults() {
	if GlobalConfig.Server.Port == 0 {
		GlobalConfig.Server.Port = 50052
	}
	if GlobalConfig.Server.Workers == 0 {
		GlobalConfig.Server.Workers = 4
	}
	if GlobalConfig.Server.QueueSize == 0 {
		GlobalConfig.Server.QueueSize = 100
	}
	if GlobalConfig.Redis.Addr == "" {
		GlobalConfig.Redis.Addr = "localhost:6379"
	}
	if GlobalConfig.Sandbox.CgroupRoot == "" {
		GlobalConfig.Sandbox.CgroupRoot = "fashoj_judger"
	}
	if GlobalConfig.Sandbox.PoolSize == 0 {
		GlobalConfig.Sandbox.PoolSize = GlobalConfig.Server.Workers
	}
	if GlobalConfig.Sandbox.MaxOutputSize == 0 {
		GlobalConfig.Sandbox.MaxOutputSize = 16 * 1024 * 1024 // 16MB
	}
	if GlobalConfig.Compilers.CPP == "" {
		GlobalConfig.Compilers.CPP = "/usr/bin/g++"
	}
	if GlobalConfig.Compilers.Python == "" {
		GlobalConfig.Compilers.Python = "/usr/bin/python3"
	}
	if GlobalConfig.Compilers.Java == "" {
		GlobalConfig.Compilers.Java = "/usr/bin/javac"
	}
}
