/* chroot.h */
#ifndef CHROOT_H
#define CHROOT_H

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <errno.h>
#include <limits.h>
#include <libgen.h>     /* dirname */

/* ---------- 工具 ---------- */
static inline int mkdir_p(const char *path, mode_t mode)
{
    char tmp[PATH_MAX];
    if (strlen(path) >= PATH_MAX) { errno = ENAMETOOLONG; return -1; }
    strncpy(tmp, path, PATH_MAX);

    for (char *p = tmp + 1; *p; ++p) {
        if (*p == '/') {
            *p = '\0';
            if (mkdir(tmp, mode) != 0 && errno != EEXIST) return -1;
            *p = '/';
        }
    }
    return (mkdir(tmp, mode) != 0 && errno != EEXIST) ? -1 : 0;
}


static inline int bind_mount(const char *src, const char *dst_root, const char *rel) {
    char dst[PATH_MAX];
    if (snprintf(dst, sizeof(dst), "%s%s", dst_root, rel) >= PATH_MAX) {
        errno = ENAMETOOLONG;
        return -1;
    }

    // 调试输出
    printf("Mounting %s -> %s\n", src, dst);

    // 确保父目录存在
    char *parent_dir = strdupa(dst); // 线程安全的复制
    parent_dir = dirname(parent_dir);
    
    if (mkdir_p(parent_dir, 0755) == -1 && errno != EEXIST) {
        perror("mkdir_p failed");
        return -1;
    }

    // 检查源文件类型
    struct stat st;
    if (lstat(src, &st) == -1) {
        perror("lstat failed");
        return -1;
    }

    // 执行挂载（禁止符号链接）
    if (mount(src, dst, NULL, MS_BIND | MS_RDONLY, NULL) == -1) {
        perror("mount failed");
        return -1;
    }

    return 0;
}

/* ---------- 主接口 ---------- */
static inline char *create_minimal_root(void) {
    // 先清理可能存在的旧目录
    // system("rm -rf /tmp/container-root-* 2>/dev/null");

    char template[] = "/tmp/container-root-XXXXXX";
    char *root = mkdtemp(template);
    if (!root) {
        perror("mkdtemp failed");
        return NULL;
    }

    printf("Created rootfs at: %s\n", root);

    // 先创建所有目录结构
    const char *dirs[] = {"/bin", "/lib", "/lib64", "/usr/lib", "/dev", "/proc", "/usr/bin", NULL};
    for (int i = 0; dirs[i]; i++) {
        char path[PATH_MAX];
        snprintf(path, sizeof(path), "%s%s", root, dirs[i]);
        if (mkdir_p(path, 0755) == -1 && errno != EEXIST) {
            perror("mkdir_p failed");
            // goto cleanup;
        }
    }

    // 按顺序挂载关键文件
    struct mount_item {
        const char *src;
        const char *dst;
        // int is_dir;
    }mounts[] = {
        {"/usr/bin/ls", "/bin/ls"},
        {"/bin/bash", "/bin/bash"},
        {"/usr/bin/bash", "/bin/bash"},  // 备用路径
        {"/lib/x86_64-linux-gnu", "/lib"},
        {"/lib64", "/lib64"},
        {NULL, NULL}
    };
    for (int i = 0; mounts[i].src; i++) {
        if (bind_mount(mounts[i].src, root, mounts[i].dst) == -1) {
            fprintf(stderr, "Failed to mount %s\n", mounts[i].src);
            // goto cleanup;
        }
    }

    // 最后挂载proc
    char proc_path[PATH_MAX];
    snprintf(proc_path, sizeof(proc_path), "%s/proc", root);
    if (mount("proc", proc_path, "proc", MS_NOSUID|MS_NODEV|MS_NOEXEC, NULL) == -1) {
        perror("mount proc failed");
        // goto cleanup;
    }

    return strdup(root);

// cleanup:
//     // 彻底清理
//     char cmd[PATH_MAX + 50];
//     snprintf(cmd, sizeof(cmd), "umount -l %s/* 2>/dev/null; rm -rf %s", root, root);
//     system(cmd);
//     return NULL;
}
#endif /* CHROOT_H */