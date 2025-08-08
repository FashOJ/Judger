#define _GNU_SOURCE  // 启用GNU扩展功能
#include <errno.h>    // 错误号处理
#include <sched.h>   // clone()系统调用
#include <stdio.h>   // 标准输入输出
#include <string.h>  // 字符串处理
#include <sys/mount.h>  // 挂载相关
#include <sys/msg.h>    // 消息队列
#include <sys/stat.h>   // 文件状态
#include <sys/types.h>  // 基本系统数据类型
#include <sys/wait.h>   // 进程等待
#include <sys/ipc.h>    // IPC相关
#include <unistd.h>     // POSIX API
#include <fcntl.h>      // open
#include <dirent.h>     //mkdir
#include "cgroup.h"      //设置cgroup
#include "chroot.h"      //创建新的根目录来隔离文件系统
#define STACKSIZE (1024 * 1024)  // 子进程栈大小(1MB)

static char stack[STACKSIZE];  // 子进程使用的栈空间

struct child_args {     //子程序输入的参数
    char *root;    /* 新生成的根目录 */
    char **argv;   /* 用户命令 */
};

// 打印错误信息
void print_err(const char *reason) {
    fprintf(stderr, "Error %s: %s\n", reason, strerror(errno));
}

// 容器内执行的函数
int container_exec(void *args) {
    struct child_args *ca = (struct child_args *)args;
    const char *newroot = ca->root;
    char **const argv   = ca->argv;
    if (mount(NULL, "/", NULL, MS_REC | MS_PRIVATE, NULL) != 0) {
        print_err("mount MS_PRIVATE");
        goto out;
    }
    //切换根目录
    if (chroot(newroot) != 0 || chdir("/") != 0) {
        print_err("chroot");
        goto out;
    }
    //生成cgroup子系统限制内存和cpu
    if(apply_cgroup_limit("mycgroup",1*1024*1024,50000, 100000)!=0){
        goto cleanup_all;
    }
    int msgid = -1;           // 消息队列ID初始化为-1
    
    /******************************************
     * 1. 重新挂载/proc文件系统
     * 使用安全标志:
     * MS_NOSUID - 禁止setuid程序
     * MS_NOEXEC - 禁止执行程序
     * MS_NODEV  - 禁止设备文件
     ******************************************/
    if (mount("proc", "/proc", "proc", MS_NOSUID|MS_NOEXEC|MS_NODEV, NULL) != 0) {
        print_err("mounting proc");
        goto out;  // 跳转到错误处理
    }

    /******************************************
     * 2. 设置新的主机名
     * 在UTS命名空间内修改，不影响主机
     ******************************************/
    const char *hostname = "new-hostname";
    if (sethostname(hostname, strlen(hostname)) != 0) {
        print_err("setting hostname");
        goto cleanup_proc;  // 先卸载/proc
    }

    /******************************************
     * 3. 创建消息队列
     * 使用ftok生成唯一key，设置权限0666
     ******************************************/
    key_t key = ftok("/tmp", 'A');  // 基于路径和ID生成key
    msgid = msgget(key, IPC_CREAT|0666);  // 创建消息队列
    if (msgid == -1) {
        print_err("creating message queue");
        goto cleanup_hostname;  // 先恢复主机名，再卸载/proc
    }

    /******************************************
     * 4. 执行用户指定的命令
     * execvp成功不会返回，失败返回-1
     ******************************************/
    if (execvp(argv[0], argv) == -1) {
        print_err("executing command");
        goto cleanup_all;  // 清理所有资源
    }

    // 正常情况下不会执行到这里
    return 0;

/******************************************
 * 错误处理部分（按资源创建逆序清理）
 ******************************************/
cleanup_all:
    /* 删除 cgroup 目录（简单实现：rmdir） */
    rmdir("/sys/fs/cgroup/mycgroup");
    if (msgid != -1) msgctl(msgid, IPC_RMID, NULL);
cleanup_hostname:
    sethostname("", 0);
cleanup_proc:
    umount("/proc");
out:
    return 1;
}
int main(int argc, char **argv) {
    // 检查命令行参数
    if (argc < 2) {
        fprintf(stderr, "Usage: %s <command> [args...]\n", argv[0]);
        return 1;
    }
    //创建容器使用的新根目录
    char *root=create_minimal_root();
    if (!root) {
        print_err("create_minimal_root");
        return 1;
    }
    struct child_args ca = { .root = root, .argv = &argv[1] };
    /******************************************
     * 设置clone的标志位，创建隔离的命名空间:
     * CLONE_NEWNET - 独立网络栈
     * CLONE_NEWUTS - 独立主机名和域名
     * CLONE_NEWNS  - 独立挂载点
     * CLONE_NEWIPC - 独立IPC
     * CLONE_NEWPID - 独立PID空间
     * CLONE_NEWUSER - 独立用户空间
     * SIGCHLD - 子进程退出时发送信号
     ******************************************/
    const int flags = CLONE_NEWNET | CLONE_NEWUTS | CLONE_NEWNS | 
                     CLONE_NEWIPC | CLONE_NEWPID | CLONE_NEWUSER | SIGCHLD;

    // 创建子进程，使用预分配的栈空间
    pid_t pid = clone(container_exec, stack + STACKSIZE, flags, &ca);
    if (pid < 0) {
        print_err("calling clone");
        return 1;
    }

    // 等待子进程结束
    int status;
    if (waitpid(pid, &status, 0) == -1) {
        print_err("waiting for pid");
        return 1;
    }
    char cmd[512];
    // snprintf(cmd, sizeof(cmd),
    //          "umount -l %s/proc 2>/dev/null; "
    //          "umount -l %s/* 2>/dev/null; "
    //          "rm -rf %s",
    //          root, root, root);
    // system(cmd); 

    free(root);
    // 返回子进程的退出状态
    return WEXITSTATUS(status);
}