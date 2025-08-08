#define _GNU_SOURCE
#include <fcntl.h>
#include <stdio.h>
#include <errno.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/types.h>
/* 在 /sys/fs/cgroup/<parent>/ 下创建子目录，并把当前 PID 写进去；
 * 然后写内存和 CPU 限制。
 * 成功返回 0，失败返回 -1 并打印错误。
 */
int apply_cgroup_limit(const char *name,
                              unsigned long mem_max_bytes,
                              unsigned long cpu_quota_us,
                              unsigned long cpu_period_us)
{
    char path[256];
    int fd;

    /* 1. 创建 cgroup 目录 */
    snprintf(path, sizeof(path), "/sys/fs/cgroup/%s", name);
    if (mkdir(path, 0755) && errno != EEXIST) {
        perror("mkdir cgroup");
        return -1;
    }

    /* 2. 把当前进程 PID 放进去 */
    snprintf(path, sizeof(path), "/sys/fs/cgroup/%s/cgroup.procs", name);
    fd = open(path, O_WRONLY);
    if (fd < 0) {
        perror("open cgroup.procs");
        return -1;
    }
    dprintf(fd, "%d\n", getpid());
    close(fd);

    /* 3. 内存限制 */
    snprintf(path, sizeof(path), "/sys/fs/cgroup/%s/memory.max", name);
    fd = open(path, O_WRONLY);
    if (fd < 0) { perror("open memory.max"); return -1; }
    dprintf(fd, "%lu\n", mem_max_bytes);
    close(fd);

    /* 4. CPU 限制 (quota/period) */
    snprintf(path, sizeof(path), "/sys/fs/cgroup/%s/cpu.max", name);
    fd = open(path, O_WRONLY);
    if (fd < 0) { perror("open cpu.max"); return -1; }
    dprintf(fd, "%lu %lu\n", cpu_quota_us, cpu_period_us);
    close(fd);

    return 0;
}
void cleanup_cgroup(const char *name)
{
    char path[256];
    snprintf(path, sizeof(path), "/sys/fs/cgroup/%s", name);
    /* 空目录才能 rmdir；忽略失败 */
    rmdir(path);
}