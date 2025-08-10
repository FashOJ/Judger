/*
 * 测试文件系统隔离功能
 * 模拟比赛中可能的文件操作和系统访问尝试
 */
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/stat.h>
#include <dirent.h>
#include <string.h>
#include <errno.h>

// 测试文件读取权限
void test_file_access() {
    printf("\n=== 测试文件访问权限 ===\n");
    
    const char* test_files[] = {
        "/etc/passwd",      // 系统密码文件
        "/etc/shadow",      // 系统影子文件
        "/proc/version",    // 系统版本信息
        "/proc/cpuinfo",    // CPU信息
        "/proc/meminfo",    // 内存信息
        "/root/.bashrc",    // 用户配置文件
        "/home",            // 用户目录
        "/tmp",             // 临时目录
        NULL
    };
    
    for (int i = 0; test_files[i]; i++) {
        printf("尝试访问: %s\n", test_files[i]);
        
        // 测试文件是否存在
        if (access(test_files[i], F_OK) == 0) {
            printf("  ✓ 文件存在\n");
            
            // 测试读权限
            if (access(test_files[i], R_OK) == 0) {
                printf("  ✓ 有读权限\n");
                
                // 尝试打开文件
                FILE *fp = fopen(test_files[i], "r");
                if (fp) {
                    printf("  ✓ 成功打开文件\n");
                    
                    // 读取前几行
                    char buffer[256];
                    int lines = 0;
                    while (fgets(buffer, sizeof(buffer), fp) && lines < 3) {
                        printf("    行%d: %s", lines + 1, buffer);
                        lines++;
                    }
                    fclose(fp);
                } else {
                    printf("  ✗ 无法打开文件: %s\n", strerror(errno));
                }
            } else {
                printf("  ✗ 无读权限: %s\n", strerror(errno));
            }
        } else {
            printf("  ✗ 文件不存在: %s\n", strerror(errno));
        }
        printf("\n");
    }
}

// 测试目录遍历
void test_directory_traversal() {
    printf("\n=== 测试目录遍历 ===\n");
    
    const char* test_dirs[] = {
        "/",
        "/bin",
        "/usr",
        "/etc",
        "/proc",
        "/sys",
        "/dev",
        NULL
    };
    
    for (int i = 0; test_dirs[i]; i++) {
        printf("尝试遍历目录: %s\n", test_dirs[i]);
        
        DIR *dir = opendir(test_dirs[i]);
        if (dir) {
            printf("  ✓ 成功打开目录\n");
            
            struct dirent *entry;
            int count = 0;
            while ((entry = readdir(dir)) && count < 10) {
                printf("    %s\n", entry->d_name);
                count++;
            }
            
            if (count == 10) {
                printf("    ... (显示前10个条目)\n");
            }
            
            closedir(dir);
        } else {
            printf("  ✗ 无法打开目录: %s\n", strerror(errno));
        }
        printf("\n");
    }
}

// 测试文件创建和写入
void test_file_creation() {
    printf("\n=== 测试文件创建权限 ===\n");
    
    const char* test_paths[] = {
        "/tmp/test_file.txt",
        "/test_file.txt",
        "/etc/test_file.txt",
        "/root/test_file.txt",
        "./local_test.txt",
        NULL
    };
    
    for (int i = 0; test_paths[i]; i++) {
        printf("尝试创建文件: %s\n", test_paths[i]);
        
        FILE *fp = fopen(test_paths[i], "w");
        if (fp) {
            printf("  ✓ 成功创建文件\n");
            
            // 写入测试数据
            if (fprintf(fp, "这是一个测试文件\n当前PID: %d\n", getpid()) > 0) {
                printf("  ✓ 成功写入数据\n");
            } else {
                printf("  ✗ 写入数据失败\n");
            }
            
            fclose(fp);
            
            // 尝试删除文件
            if (unlink(test_paths[i]) == 0) {
                printf("  ✓ 成功删除文件\n");
            } else {
                printf("  ✗ 删除文件失败: %s\n", strerror(errno));
            }
        } else {
            printf("  ✗ 无法创建文件: %s\n", strerror(errno));
        }
        printf("\n");
    }
}

// 测试符号链接和硬链接
void test_links() {
    printf("\n=== 测试链接操作 ===\n");
    
    // 创建测试文件
    const char* source = "./source.txt";
    const char* symlink_path = "./test_symlink";
    const char* hardlink_path = "./test_hardlink";
    
    FILE *fp = fopen(source, "w");
    if (fp) {
        fprintf(fp, "测试链接文件\n");
        fclose(fp);
        printf("✓ 创建源文件成功\n");
        
        // 测试符号链接
        if (symlink(source, symlink_path) == 0) {
            printf("✓ 创建符号链接成功\n");
            unlink(symlink_path);
        } else {
            printf("✗ 创建符号链接失败: %s\n", strerror(errno));
        }
        
        // 测试硬链接
        if (link(source, hardlink_path) == 0) {
            printf("✓ 创建硬链接成功\n");
            unlink(hardlink_path);
        } else {
            printf("✗ 创建硬链接失败: %s\n", strerror(errno));
        }
        
        unlink(source);
    } else {
        printf("✗ 无法创建源文件\n");
    }
}

int main() {
    printf("=== 文件系统隔离测试 ===\n");
    printf("PID: %d\n", getpid());
    printf("当前工作目录: %s\n", getcwd(NULL, 0));
    
    test_file_access();
    test_directory_traversal();
    test_file_creation();
    test_links();
    
    printf("\n=== 文件系统隔离测试完成 ===\n");
    return 0;
}