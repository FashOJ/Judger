package sandbox

import (
	libseccomp "github.com/seccomp/libseccomp-golang"
)

// LoadSeccompProfile 加载 Seccomp 规则
// 这是一个默认的安全策略，允许大多数无害的 syscall，禁止危险操作
func LoadSeccompProfile() (*libseccomp.ScmpFilter, error) {
	// 默认动作：Kill (如果违反规则直接杀死)
	// 也可以选择 ERRNO 返回错误
	filter, err := libseccomp.NewFilter(libseccomp.ActKill)
	if err != nil {
		return nil, err
	}

	// 白名单：允许必要的 syscall
	// 这是一个基础列表，根据实际需要可能要补充
	allowList := []string{
		"read", "write", "readv", "writev", "close", "fstat", "lseek", "dup", "dup2", "dup3",
		"mmap", "mprotect", "munmap", "brk", "mremap", "msync", "mincore", "madvise",
		"rt_sigaction", "rt_sigprocmask", "rt_sigreturn", "rt_sigpending",
		"sigaltstack", "restart_syscall", "clone", "execve", "exit", "exit_group",
		"arch_prctl", "set_tid_address", "set_robust_list", "sysinfo", "uname", "times",
		"futex", "getrlimit", "getuid", "getgid", "geteuid", "getegid", "getppid", "getpgrp",
		"getpid", "gettid", "capget", "capset", "prlimit64",
		"stat", "lstat", "fstat", "newfstatat", // 文件状态
		"access", "faccessat",
		"open", "openat", // 打开文件 (需要配合路径限制，但 seccomp 只管 syscall)
		"fcntl", "ioctl",
		"getcwd", "readlink", "readlinkat",
		"gettimeofday", "clock_gettime", "clock_getres", "clock_nanosleep",
		// 内存分配相关
		"mbind", "get_mempolicy", "set_mempolicy",
	}

	for _, name := range allowList {
		syscallID, err := libseccomp.GetSyscallFromName(name)
		if err != nil {
			// 某些架构可能不支持某些 syscall，忽略
			continue
		}
		// 允许这些 syscall
		if err := filter.AddRule(syscallID, libseccomp.ActAllow); err != nil {
			return nil, err
		}
	}

	return filter, nil
}
