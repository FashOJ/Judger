/*
 * 测试系统调用限制功能
 * 模拟可能使用的各种系统调用
 */
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/syscall.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <sys/mount.h>
#include <sys/socket.h>
#include <sys/ptrace.h>
#include <sys/reboot.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>
#include <time.h>
#include <sys/time.h>
#include <sys/utsname.h>
#include <sched.h>

// 测试基本系统调用
void test_basic_syscalls() {
    printf("\n=== 测试基本系统调用 ===\n");
    
    // 测试时间相关系统调用
    printf("1. 时间相关系统调用:\n");
    
    time_t current_time = time(NULL);
    if (current_time != (time_t)-1) {
        printf("  ✓ time(): %s", ctime(&current_time));
    } else {
        printf("  ✗ time() 失败: %s\n", strerror(errno));
    }
    
    struct timeval tv;
    if (gettimeofday(&tv, NULL) == 0) {
        printf("  ✓ gettimeofday(): %ld.%06ld\n", tv.tv_sec, tv.tv_usec);
    } else {
        printf("  ✗ gettimeofday() 失败: %s\n", strerror(errno));
    }
    
    struct timespec ts;
    if (clock_gettime(CLOCK_REALTIME, &ts) == 0) {
        printf("  ✓ clock_gettime(): %ld.%09ld\n", ts.tv_sec, ts.tv_nsec);
    } else {
        printf("  ✗ clock_gettime() 失败: %s\n", strerror(errno));
    }
    
    // 测试进程信息系统调用
    printf("\n2. 进程信息系统调用:\n");
    
    printf("  ✓ getpid(): %d\n", getpid());
    printf("  ✓ getppid(): %d\n", getppid());
    printf("  ✓ getuid(): %d\n", getuid());
    printf("  ✓ getgid(): %d\n", getgid());
    
    struct utsname uts;
    if (uname(&uts) == 0) {
        printf("  ✓ uname(): %s %s %s\n", uts.sysname, uts.release, uts.machine);
    } else {
        printf("  ✗ uname() 失败: %s\n", strerror(errno));
    }
}

// 测试文件系统相关系统调用
void test_filesystem_syscalls() {
    printf("\n=== 测试文件系统系统调用 ===\n");
    
    // 测试文件操作
    printf("1. 文件操作系统调用:\n");
    
    const char *test_file = "./syscall_test.txt";
    
    // open
    int fd = open(test_file, O_CREAT | O_WRONLY | O_TRUNC, 0644);
    if (fd >= 0) {
        printf("  ✓ open() 成功，fd: %d\n", fd);
        
        // write
        const char *data = "测试数据\n";
        ssize_t written = write(fd, data, strlen(data));
        if (written > 0) {
            printf("  ✓ write() 成功，写入 %zd 字节\n", written);
        } else {
            printf("  ✗ write() 失败: %s\n", strerror(errno));
        }
        
        // fsync
        if (fsync(fd) == 0) {
            printf("  ✓ fsync() 成功\n");
        } else {
            printf("  ✗ fsync() 失败: %s\n", strerror(errno));
        }
        
        close(fd);
        printf("  ✓ close() 成功\n");
        
        // stat
        struct stat st;
        if (stat(test_file, &st) == 0) {
            printf("  ✓ stat() 成功，文件大小: %ld 字节\n", st.st_size);
        } else {
            printf("  ✗ stat() 失败: %s\n", strerror(errno));
        }
        
        // unlink
        if (unlink(test_file) == 0) {
            printf("  ✓ unlink() 成功\n");
        } else {
            printf("  ✗ unlink() 失败: %s\n", strerror(errno));
        }
    } else {
        printf("  ✗ open() 失败: %s\n", strerror(errno));
    }
    
    // 测试目录操作
    printf("\n2. 目录操作系统调用:\n");
    
    const char *test_dir = "./test_dir";
    
    if (mkdir(test_dir, 0755) == 0) {
        printf("  ✓ mkdir() 成功\n");
        
        if (rmdir(test_dir) == 0) {
            printf("  ✓ rmdir() 成功\n");
        } else {
            printf("  ✗ rmdir() 失败: %s\n", strerror(errno));
        }
    } else {
        printf("  ✗ mkdir() 失败: %s\n", strerror(errno));
    }
}

// 测试危险系统调用
void test_dangerous_syscalls() {
    printf("\n=== 测试危险系统调用 ===\n");
    
    // 测试mount（应该被阻止）
    printf("1. 测试mount系统调用:\n");
    if (mount("none", "/tmp/test_mount", "tmpfs", 0, NULL) == 0) {
        printf("  ⚠️  mount() 成功 - 这可能是安全风险\n");
        umount("/tmp/test_mount");
    } else {
        printf("  ✓ mount() 被阻止: %s\n", strerror(errno));
    }
    
    // 测试ptrace（应该被阻止）
    printf("\n2. 测试ptrace系统调用:\n");
    if (ptrace(PTRACE_TRACEME, 0, NULL, NULL) == 0) {
        printf("  ⚠️  ptrace() 成功 - 这可能是安全风险\n");
    } else {
        printf("  ✓ ptrace() 被阻止: %s\n", strerror(errno));
    }
    
    // 测试reboot（应该被阻止）
    printf("\n3. 测试reboot系统调用:\n");
    if (reboot(RB_AUTOBOOT) == 0) {
        printf("  ⚠️  reboot() 成功 - 系统可能重启\n");
    } else {
        printf("  ✓ reboot() 被阻止: %s\n", strerror(errno));
    }
    
    // 测试socket创建（可能被限制）
    printf("\n4. 测试socket系统调用:\n");
    int sockfd = socket(AF_INET, SOCK_STREAM, 0);
    if (sockfd >= 0) {
        printf("  ⚠️  socket() 成功 - 网络访问可能可用\n");
        close(sockfd);
    } else {
        printf("  ✓ socket() 被阻止: %s\n", strerror(errno));
    }
    
    // 测试clone（可能被限制）
    printf("\n5. 测试clone系统调用:\n");
    pid_t clone_pid = syscall(SYS_clone, SIGCHLD, NULL);
    if (clone_pid == 0) {
        printf("  ⚠️  clone() 成功 - 子进程创建\n");
        _exit(0);
    } else if (clone_pid > 0) {
        printf("  ⚠️  clone() 成功 - 创建了子进程 %d\n", clone_pid);
        int status;
        waitpid(clone_pid, &status, 0);
    } else {
        printf("  ✓ clone() 被阻止: %s\n", strerror(errno));
    }
}

// 测试内存相关系统调用
void test_memory_syscalls() {
    printf("\n=== 测试内存相关系统调用 ===\n");
    
    // 测试brk
    printf("1. 测试brk系统调用:\n");
    void *current_brk = sbrk(0);
    printf("  当前brk: %p\n", current_brk);
    
    void *new_brk = sbrk(4096); // 增加4KB
    if (new_brk != (void*)-1) {
        printf("  ✓ sbrk() 成功，新brk: %p\n", sbrk(0));
        sbrk(-4096); // 恢复
    } else {
        printf("  ✗ sbrk() 失败: %s\n", strerror(errno));
    }
    
    // 测试mmap
    printf("\n2. 测试mmap系统调用:\n");
    void *mapped = mmap(NULL, 4096, PROT_READ | PROT_WRITE, 
                       MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    if (mapped != MAP_FAILED) {
        printf("  ✓ mmap() 成功，地址: %p\n", mapped);
        
        // 写入测试数据
        *(int*)mapped = 0x12345678;
        printf("  ✓ 内存写入成功: 0x%x\n", *(int*)mapped);
        
        if (munmap(mapped, 4096) == 0) {
            printf("  ✓ munmap() 成功\n");
        } else {
            printf("  ✗ munmap() 失败: %s\n", strerror(errno));
        }
    } else {
        printf("  ✗ mmap() 失败: %s\n", strerror(errno));
    }
}

// 测试直接系统调用
void test_direct_syscalls() {
    printf("\n=== 测试直接系统调用 ===\n");
    
    // 使用syscall()直接调用
    printf("1. 直接系统调用测试:\n");
    
    // 直接调用getpid
    pid_t pid = syscall(SYS_getpid);
    printf("  ✓ syscall(SYS_getpid): %d\n", pid);
    
    // 直接调用gettid
    pid_t tid = syscall(SYS_gettid);
    printf("  ✓ syscall(SYS_gettid): %d\n", tid);
    
    // 尝试一些可能被过滤的系统调用
    printf("\n2. 可能被过滤的系统调用:\n");
    
    // 尝试execve
    printf("  测试 SYS_execve...\n");
    // 注意：这里不实际执行，只是测试是否被过滤
    
    // 尝试socket
    int sock_result = syscall(SYS_socket, AF_INET, SOCK_STREAM, 0);
    if (sock_result >= 0) {
        printf("  ⚠️  syscall(SYS_socket) 成功: %d\n", sock_result);
        close(sock_result);
    } else {
        printf("  ✓ syscall(SYS_socket) 被阻止: %s\n", strerror(errno));
    }
    
    // 尝试kill
    int kill_result = syscall(SYS_kill, getpid(), 0); // 发送空信号测试
    if (kill_result == 0) {
        printf("  ✓ syscall(SYS_kill) 成功\n");
    } else {
        printf("  ✗ syscall(SYS_kill) 失败: %s\n", strerror(errno));
    }
}

int main() {
    printf("=== 系统调用限制测试 ===\n");
    printf("PID: %d\n", getpid());
    printf("测试各种系统调用是否被正确限制...\n");
    
    test_basic_syscalls();
    test_filesystem_syscalls();
    test_memory_syscalls();
    test_dangerous_syscalls();
    test_direct_syscalls();
    
    printf("\n=== 系统调用限制测试完成 ===\n");
    printf("注意：\n");
    printf("  ✓ 表示操作成功或被正确阻止\n");
    printf("  ✗ 表示操作失败\n");
    printf("  ⚠️  表示潜在的安全风险\n");
    
    return 0;
}