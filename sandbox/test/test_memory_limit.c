/*
 * 测试内存限制功能
 * 模拟中内存超限的情况
 */
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/wait.h>
#include <errno.h>

// 模拟一个会消耗大量内存的算法竞赛程序
void memory_intensive_program() {
    printf("开始内存密集型程序...\n");
    
    // 模拟动态规划算法，分配大量内存
    size_t size = 10 * 1024 * 1024; // 10MB
    char **arrays[100];
    
    for (int i = 0; i < 100; i++) {
        arrays[i] = malloc(size);
        if (arrays[i] == NULL) {
            printf("内存分配失败在第 %d 次尝试\n", i);
            printf("错误: %s\n", strerror(errno));
            break;
        }
        
        // 实际写入内存以确保分配
        memset(arrays[i], 'A' + (i % 26), size);
        printf("成功分配第 %d 块内存 (%zu MB)\n", i + 1, size / (1024 * 1024));
        
        // 模拟算法处理时间
        usleep(100000); // 100ms
    }
    
    printf("程序正常结束\n");
}

int main() {
    printf("=== 内存限制测试 ===\n");
    printf("PID: %d\n", getpid());
    printf("当前进程将尝试分配大量内存...\n");
    
    memory_intensive_program();
    
    return 0;
}