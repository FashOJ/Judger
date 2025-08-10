/*
 * 测试进程隔离功能
 * 模拟可能的进程操作和系统调用
 */
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/wait.h>
#include <sys/types.h>
#include <signal.h>
#include <errno.h>
#include <string.h>
#include <dirent.h>
#include <fcntl.h>
#include <sys/stat.h>
#include <sys/resource.h>

// 测试进程创建
void test_process_creation() {
    printf("\n=== 测试进程创建 ===\n");
    printf("当前PID: %d, PPID: %d\n", getpid(), getppid());
    
    // 测试fork
    printf("\n1. 测试fork():\n");
    pid_t fork_pid = fork();
    if (fork_pid == 0) {
        // 子进程
        printf("  ✓ fork成功 - 子进程PID: %d, PPID: %d\n", getpid(), getppid());
        
        // 子进程执行一些简单任务
        printf("  子进程正在执行任务...\n");
        for (int i = 0; i < 3; i++) {
            printf("    子进程计数: %d\n", i + 1);
            sleep(1);
        }
        printf("  子进程任务完成\n");
        exit(0);
    } else if (fork_pid > 0) {
        // 父进程
        printf("  ✓ fork成功 - 创建子进程PID: %d\n", fork_pid);
        
        int status;
        if (waitpid(fork_pid, &status, 0) > 0) {
            printf("  ✓ 子进程正常结束，退出码: %d\n", WEXITSTATUS(status));
        }
    } else {
        printf("  ✗ fork失败: %s\n", strerror(errno));
    }
    
    // 测试vfork
    printf("\n2. 测试vfork():\n");
    pid_t vfork_pid = vfork();
    if (vfork_pid == 0) {
        printf("  ✓ vfork成功 - 子进程PID: %d\n", getpid());
        _exit(0); // vfork必须使用_exit
    } else if (vfork_pid > 0) {
        printf("  ✓ vfork成功 - 创建子进程PID: %d\n", vfork_pid);
    } else {
        printf("  ✗ vfork失败: %s\n", strerror(errno));
    }
    
    // 测试execve
    printf("\n3. 测试execve():\n");
    pid_t exec_pid = fork();
    if (exec_pid == 0) {
        printf("  尝试执行 /bin/echo...\n");
        char *args[] = {"/bin/echo", "Hello from execve!", NULL};
        char *env[] = {NULL};
        
        if (execve("/bin/echo", args, env) == -1) {
            printf("  ✗ execve失败: %s\n", strerror(errno));
            _exit(1);
        }
    } else if (exec_pid > 0) {
        int status;
        waitpid(exec_pid, &status, 0);
        if (WIFEXITED(status) && WEXITSTATUS(status) == 0) {
            printf("  ✓ execve执行成功\n");
        } else {
            printf("  ✗ execve执行失败\n");
        }
    }
}

// 测试进程信息访问
void test_process_info() {
    printf("\n=== 测试进程信息访问 ===\n");
    
    // 读取/proc/self信息
    printf("1. 读取当前进程信息:\n");
    
    const char *proc_files[] = {
        "/proc/self/status",
        "/proc/self/cmdline",
        "/proc/self/environ",
        "/proc/self/maps",
        "/proc/self/limits",
        NULL
    };
    
    for (int i = 0; proc_files[i]; i++) {
        printf("  读取 %s:\n", proc_files[i]);
        
        FILE *fp = fopen(proc_files[i], "r");
        if (fp) {
            char buffer[256];
            int lines = 0;
            while (fgets(buffer, sizeof(buffer), fp) && lines < 5) {
                printf("    %s", buffer);
                lines++;
            }
            if (lines == 5) {
                printf("    ... (显示前5行)\n");
            }
            fclose(fp);
            printf("  ✓ 读取成功\n\n");
        } else {
            printf("  ✗ 读取失败: %s\n\n", strerror(errno));
        }
    }
    
    // 尝试访问其他进程信息
    printf("2. 尝试访问其他进程信息:\n");
    
    DIR *proc_dir = opendir("/proc");
    if (proc_dir) {
        struct dirent *entry;
        int count = 0;
        
        while ((entry = readdir(proc_dir)) && count < 5) {
            // 检查是否是数字目录（进程ID）
            if (strspn(entry->d_name, "0123456789") == strlen(entry->d_name)) {
                int pid = atoi(entry->d_name);
                if (pid != getpid() && pid > 1) { // 不是当前进程且不是init
                    char path[256];
                    snprintf(path, sizeof(path), "/proc/%d/cmdline", pid);
                    
                    FILE *fp = fopen(path, "r");
                    if (fp) {
                        char cmdline[256] = {0};
                        fread(cmdline, 1, sizeof(cmdline) - 1, fp);
                        // 替换null字符为空格
                        for (int i = 0; cmdline[i]; i++) {
                            if (cmdline[i] == '\0') cmdline[i] = ' ';
                        }
                        printf("  PID %d: %s\n", pid, cmdline[0] ? cmdline : "<unknown>");
                        fclose(fp);
                        count++;
                    }
                }
            }
        }
        closedir(proc_dir);
        
        if (count > 0) {
            printf("  ✓ 可以访问其他进程信息\n");
        } else {
            printf("  ✗ 无法访问其他进程信息\n");
        }
    }
}

// 测试信号处理
void signal_handler(int sig) {
    printf("  收到信号 %d (%s)\n", sig, strsignal(sig));
}

void test_signal_handling() {
    printf("\n=== 测试信号处理 ===\n");
    
    // 注册信号处理器
    signal(SIGUSR1, signal_handler);
    signal(SIGUSR2, signal_handler);
    signal(SIGTERM, signal_handler);
    
    printf("1. 自发信号测试:\n");
    printf("  发送SIGUSR1给自己...\n");
    kill(getpid(), SIGUSR1);
    
    printf("  发送SIGUSR2给自己...\n");
    kill(getpid(), SIGUSR2);
    
    // 测试向其他进程发送信号
    printf("\n2. 向其他进程发送信号测试:\n");
    
    // 尝试向init进程发送信号
    printf("  尝试向PID 1发送SIGUSR1...\n");
    if (kill(1, SIGUSR1) == 0) {
        printf("  ✓ 信号发送成功\n");
    } else {
        printf("  ✗ 信号发送失败: %s\n", strerror(errno));
    }
    
    // 尝试向不存在的进程发送信号
    printf("  尝试向PID 99999发送信号...\n");
    if (kill(99999, SIGUSR1) == 0) {
        printf("  ✓ 信号发送成功\n");
    } else {
        printf("  ✗ 信号发送失败: %s\n", strerror(errno));
    }
}

// 测试资源限制
void test_resource_limits() {
    printf("\n=== 测试资源限制 ===\n");
    
    struct rlimit limit;
    
    const struct {
        int resource;
        const char *name;
    } resources[] = {
        {RLIMIT_CPU, "CPU时间"},
        {RLIMIT_FSIZE, "文件大小"},
        {RLIMIT_DATA, "数据段大小"},
        {RLIMIT_STACK, "栈大小"},
        {RLIMIT_CORE, "核心转储大小"},
        {RLIMIT_RSS, "常驻内存大小"},
        {RLIMIT_NPROC, "进程数量"},
        {RLIMIT_NOFILE, "文件描述符数量"},
        {RLIMIT_MEMLOCK, "锁定内存大小"},
        {RLIMIT_AS, "虚拟内存大小"},
        {-1, NULL}
    };
    
    for (int i = 0; resources[i].resource != -1; i++) {
        if (getrlimit(resources[i].resource, &limit) == 0) {
            printf("  %s:\n", resources[i].name);
            printf("    软限制: ");
            if (limit.rlim_cur == RLIM_INFINITY) {
                printf("无限制\n");
            } else {
                printf("%lu\n", limit.rlim_cur);
            }
            printf("    硬限制: ");
            if (limit.rlim_max == RLIM_INFINITY) {
                printf("无限制\n");
            } else {
                printf("%lu\n", limit.rlim_max);
            }
        } else {
            printf("  %s: 获取失败 - %s\n", resources[i].name, strerror(errno));
        }
    }
}

int main() {
    printf("=== 进程隔离测试 ===\n");
    printf("PID: %d\n", getpid());
    printf("PPID: %d\n", getppid());
    printf("UID: %d, GID: %d\n", getuid(), getgid());
    printf("EUID: %d, EGID: %d\n", geteuid(), getegid());
    
    test_process_creation();
    test_process_info();
    test_signal_handling();
    test_resource_limits();
    
    printf("\n=== 进程隔离测试完成 ===\n");
    return 0;
}