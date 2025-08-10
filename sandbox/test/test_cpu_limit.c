/*
 * 测试CPU限制功能
 * 模拟CPU密集型计算
 */
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <time.h>
#include <sys/time.h>
#include <math.h>

// 模拟素数筛选算法（埃拉托斯特尼筛法）
void sieve_of_eratosthenes(int n) {
    printf("开始计算前 %d 个数的素数...\n", n);
    
    int *prime = calloc(n + 1, sizeof(int));
    if (!prime) {
        printf("内存分配失败\n");
        return;
    }
    
    // 初始化
    for (int i = 2; i <= n; i++) {
        prime[i] = 1;
    }
    
    // 筛选素数
    for (int p = 2; p * p <= n; p++) {
        if (prime[p] == 1) {
            for (int i = p * p; i <= n; i += p) {
                prime[i] = 0;
            }
        }
    }
    
    // 统计素数个数
    int count = 0;
    for (int i = 2; i <= n; i++) {
        if (prime[i]) {
            count++;
        }
    }
    
    printf("找到 %d 个素数\n", count);
    free(prime);
}

// 模拟矩阵乘法运算
void matrix_multiplication(int size) {
    printf("开始 %dx%d 矩阵乘法运算...\n", size, size);
    
    double **a = malloc(size * sizeof(double*));
    double **b = malloc(size * sizeof(double*));
    double **c = malloc(size * sizeof(double*));
    
    for (int i = 0; i < size; i++) {
        a[i] = malloc(size * sizeof(double));
        b[i] = malloc(size * sizeof(double));
        c[i] = malloc(size * sizeof(double));
    }
    
    // 初始化矩阵
    for (int i = 0; i < size; i++) {
        for (int j = 0; j < size; j++) {
            a[i][j] = rand() % 100;
            b[i][j] = rand() % 100;
            c[i][j] = 0;
        }
    }
    
    // 矩阵乘法
    struct timeval start, end;
    gettimeofday(&start, NULL);
    
    for (int i = 0; i < size; i++) {
        for (int j = 0; j < size; j++) {
            for (int k = 0; k < size; k++) {
                c[i][j] += a[i][k] * b[k][j];
            }
        }
    }
    
    gettimeofday(&end, NULL);
    double time_used = (end.tv_sec - start.tv_sec) + (end.tv_usec - start.tv_usec) / 1000000.0;
    printf("矩阵乘法完成，耗时: %.2f 秒\n", time_used);
    
    // 清理内存
    for (int i = 0; i < size; i++) {
        free(a[i]);
        free(b[i]);
        free(c[i]);
    }
    free(a);
    free(b);
    free(c);
}

// 模拟递归斐波那契计算（低效版本，用于消耗CPU）
long long fibonacci_recursive(int n) {
    if (n <= 1) return n;
    return fibonacci_recursive(n - 1) + fibonacci_recursive(n - 2);
}

int main() {
    printf("=== CPU限制测试 ===\n");
    printf("PID: %d\n", getpid());
    
    struct timeval start, end;
    gettimeofday(&start, NULL);
    
    // 测试1: 素数筛选
    sieve_of_eratosthenes(1000000);
    
    // 测试2: 矩阵乘法
    matrix_multiplication(500);
    
    // 测试3: 递归斐波那契（CPU密集型）
    printf("开始计算斐波那契数列...\n");
    for (int i = 35; i <= 40; i++) {
        long long result = fibonacci_recursive(i);
        printf("fibonacci(%d) = %lld\n", i, result);
    }
    
    gettimeofday(&end, NULL);
    double total_time = (end.tv_sec - start.tv_sec) + (end.tv_usec - start.tv_usec) / 1000000.0;
    printf("总执行时间: %.2f 秒\n", total_time);
    
    return 0;
}