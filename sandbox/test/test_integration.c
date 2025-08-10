/*
 * 沙箱集成测试
 * 模拟真实的算法竞赛程序，测试沙箱的综合性能
 */
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <time.h>
#include <sys/time.h>
#include <sys/resource.h>
#include <errno.h>
#include <math.h>

// 模拟ACM竞赛题目1：大数斐波那契
void test_fibonacci_large() {
    printf("\n=== 测试1: 大数斐波那契计算 ===\n");
    printf("模拟ACM题目：计算第N个斐波那契数（大数运算）\n");
    
    struct timeval start, end;
    gettimeofday(&start, NULL);
    
    const int N = 10000;
    
    // 使用数组模拟大数
    int *fib_prev = calloc(1000, sizeof(int));
    int *fib_curr = calloc(1000, sizeof(int));
    int *fib_next = calloc(1000, sizeof(int));
    
    if (!fib_prev || !fib_curr || !fib_next) {
        printf("内存分配失败\n");
        return;
    }
    
    fib_prev[0] = 0;
    fib_curr[0] = 1;
    
    printf("计算第 %d 个斐波那契数...\n", N);
    
    for (int i = 2; i <= N; i++) {
        // 大数加法
        int carry = 0;
        for (int j = 0; j < 1000; j++) {
            int sum = fib_prev[j] + fib_curr[j] + carry;
            fib_next[j] = sum % 10;
            carry = sum / 10;
        }
        
        // 交换数组
        int *temp = fib_prev;
        fib_prev = fib_curr;
        fib_curr = fib_next;
        fib_next = temp;
        
        memset(fib_next, 0, 1000 * sizeof(int));
        
        if (i % 1000 == 0) {
            printf("  进度: %d/%d\n", i, N);
        }
    }
    
    // 输出结果的前20位和后20位
    printf("结果前20位: ");
    int start_pos = 999;
    while (start_pos > 0 && fib_curr[start_pos] == 0) start_pos--;
    
    for (int i = start_pos; i >= start_pos - 19 && i >= 0; i--) {
        printf("%d", fib_curr[i]);
    }
    printf("...后20位: ");
    for (int i = 19; i >= 0; i--) {
        printf("%d", fib_curr[i]);
    }
    printf("\n");
    
    gettimeofday(&end, NULL);
    double time_used = (end.tv_sec - start.tv_sec) + (end.tv_usec - start.tv_usec) / 1000000.0;
    printf("计算完成，耗时: %.3f 秒\n", time_used);
    
    free(fib_prev);
    free(fib_curr);
    free(fib_next);
}

// 模拟ACM竞赛题目2：图论最短路径
void test_shortest_path() {
    printf("\n=== 测试2: 图论最短路径算法 ===\n");
    printf("模拟ACM题目：Dijkstra算法求最短路径\n");
    
    struct timeval start, end;
    gettimeofday(&start, NULL);
    
    const int N = 1000; // 节点数
    const int INF = 1000000;
    
    // 分配邻接矩阵
    int **graph = malloc(N * sizeof(int*));
    int *dist = malloc(N * sizeof(int));
    int *visited = malloc(N * sizeof(int));
    
    for (int i = 0; i < N; i++) {
        graph[i] = malloc(N * sizeof(int));
    }
    
    if (!graph || !dist || !visited) {
        printf("内存分配失败\n");
        return;
    }
    
    printf("生成随机图...\n");
    srand(time(NULL));
    
    // 初始化图
    for (int i = 0; i < N; i++) {
        for (int j = 0; j < N; j++) {
            if (i == j) {
                graph[i][j] = 0;
            } else if (rand() % 100 < 20) { // 20%的概率有边
                graph[i][j] = rand() % 100 + 1;
            } else {
                graph[i][j] = INF;
            }
        }
    }
    
    printf("执行Dijkstra算法...\n");
    
    // Dijkstra算法
    for (int i = 0; i < N; i++) {
        dist[i] = INF;
        visited[i] = 0;
    }
    dist[0] = 0;
    
    for (int count = 0; count < N - 1; count++) {
        int min_dist = INF, min_index = -1;
        
        // 找到最小距离的未访问节点
        for (int v = 0; v < N; v++) {
            if (!visited[v] && dist[v] < min_dist) {
                min_dist = dist[v];
                min_index = v;
            }
        }
        
        if (min_index == -1) break;
        
        visited[min_index] = 1;
        
        // 更新相邻节点的距离
        for (int v = 0; v < N; v++) {
            if (!visited[v] && graph[min_index][v] != INF &&
                dist[min_index] + graph[min_index][v] < dist[v]) {
                dist[v] = dist[min_index] + graph[min_index][v];
            }
        }
        
        if (count % 100 == 0) {
            printf("  进度: %d/%d\n", count, N - 1);
        }
    }
    
    // 输出部分结果
    printf("从节点0到其他节点的最短距离（前10个）:\n");
    for (int i = 1; i <= 10 && i < N; i++) {
        if (dist[i] == INF) {
            printf("  到节点%d: 不可达\n", i);
        } else {
            printf("  到节点%d: %d\n", i, dist[i]);
        }
    }
    
    gettimeofday(&end, NULL);
    double time_used = (end.tv_sec - start.tv_sec) + (end.tv_usec - start.tv_usec) / 1000000.0;
    printf("算法完成，耗时: %.3f 秒\n", time_used);
    
    // 清理内存
    for (int i = 0; i < N; i++) {
        free(graph[i]);
    }
    free(graph);
    free(dist);
    free(visited);
}

// 模拟ACM竞赛题目3：动态规划背包问题
void test_knapsack() {
    printf("\n=== 测试3: 动态规划背包问题 ===\n");
    printf("模拟ACM题目：0-1背包问题\n");
    
    struct timeval start, end;
    gettimeofday(&start, NULL);
    
    const int N = 1000;  // 物品数量
    const int W = 5000;  // 背包容量
    
    int *weights = malloc(N * sizeof(int));
    int *values = malloc(N * sizeof(int));
    int **dp = malloc((N + 1) * sizeof(int*));
    
    for (int i = 0; i <= N; i++) {
        dp[i] = malloc((W + 1) * sizeof(int));
    }
    
    if (!weights || !values || !dp) {
        printf("内存分配失败\n");
        return;
    }
    
    printf("生成随机物品数据...\n");
    srand(time(NULL) + 1);
    
    for (int i = 0; i < N; i++) {
        weights[i] = rand() % 50 + 1;  // 重量1-50
        values[i] = rand() % 100 + 1;  // 价值1-100
    }
    
    printf("执行动态规划算法...\n");
    
    // 初始化DP表
    for (int i = 0; i <= N; i++) {
        for (int w = 0; w <= W; w++) {
            dp[i][w] = 0;
        }
    }
    
    // 填充DP表
    for (int i = 1; i <= N; i++) {
        for (int w = 1; w <= W; w++) {
            if (weights[i-1] <= w) {
                int include = values[i-1] + dp[i-1][w - weights[i-1]];
                int exclude = dp[i-1][w];
                dp[i][w] = (include > exclude) ? include : exclude;
            } else {
                dp[i][w] = dp[i-1][w];
            }
        }
        
        if (i % 100 == 0) {
            printf("  进度: %d/%d\n", i, N);
        }
    }
    
    printf("最大价值: %d\n", dp[N][W]);
    
    // 回溯找出选择的物品
    printf("选择的物品（前10个）:\n");
    int w = W, count = 0;
    for (int i = N; i > 0 && w > 0 && count < 10; i--) {
        if (dp[i][w] != dp[i-1][w]) {
            printf("  物品%d: 重量=%d, 价值=%d\n", i, weights[i-1], values[i-1]);
            w -= weights[i-1];
            count++;
        }
    }
    
    gettimeofday(&end, NULL);
    double time_used = (end.tv_sec - start.tv_sec) + (end.tv_usec - start.tv_usec) / 1000000.0;
    printf("算法完成，耗时: %.3f 秒\n", time_used);
    
    // 清理内存
    free(weights);
    free(values);
    for (int i = 0; i <= N; i++) {
        free(dp[i]);
    }
    free(dp);
}

// 模拟文件I/O密集型操作
void test_file_io() {
    printf("\n=== 测试4: 文件I/O操作 ===\n");
    printf("模拟需要大量文件读写的算法\n");
    
    struct timeval start, end;
    gettimeofday(&start, NULL);
    
    const char *filename = "./test_data.txt";
    const int DATA_SIZE = 100000;
    
    printf("写入测试数据...\n");
    
    // 写入大量数据
    FILE *fp = fopen(filename, "w");
    if (!fp) {
        printf("无法创建文件: %s\n", strerror(errno));
        return;
    }
    
    for (int i = 0; i < DATA_SIZE; i++) {
        fprintf(fp, "%d %d %d\n", i, i * 2, i * 3);
        if (i % 10000 == 0) {
            printf("  写入进度: %d/%d\n", i, DATA_SIZE);
        }
    }
    fclose(fp);
    
    printf("读取并处理数据...\n");
    
    // 读取并处理数据
    fp = fopen(filename, "r");
    if (!fp) {
        printf("无法打开文件: %s\n", strerror(errno));
        return;
    }
    
    long long sum = 0;
    int a, b, c, count = 0;
    
    while (fscanf(fp, "%d %d %d", &a, &b, &c) == 3) {
        sum += a + b + c;
        count++;
        
        if (count % 10000 == 0) {
            printf("  读取进度: %d/%d\n", count, DATA_SIZE);
        }
    }
    fclose(fp);
    
    printf("处理了 %d 行数据，总和: %lld\n", count, sum);
    
    // 删除测试文件
    if (unlink(filename) == 0) {
        printf("清理测试文件成功\n");
    }
    
    gettimeofday(&end, NULL);
    double time_used = (end.tv_sec - start.tv_sec) + (end.tv_usec - start.tv_usec) / 1000000.0;
    printf("I/O操作完成，耗时: %.3f 秒\n", time_used);
}

// 获取资源使用情况
void print_resource_usage() {
    printf("\n=== 资源使用情况 ===\n");
    
    struct rusage usage;
    if (getrusage(RUSAGE_SELF, &usage) == 0) {
        printf("用户CPU时间: %ld.%06ld 秒\n", 
               usage.ru_utime.tv_sec, usage.ru_utime.tv_usec);
        printf("系统CPU时间: %ld.%06ld 秒\n", 
               usage.ru_stime.tv_sec, usage.ru_stime.tv_usec);
        printf("最大常驻内存: %ld KB\n", usage.ru_maxrss);
        printf("页面错误次数: %ld\n", usage.ru_majflt);
        printf("文件系统输入: %ld\n", usage.ru_inblock);
        printf("文件系统输出: %ld\n", usage.ru_oublock);
        printf("上下文切换(主动): %ld\n", usage.ru_nvcsw);
        printf("上下文切换(被动): %ld\n", usage.ru_nivcsw);
    } else {
        printf("无法获取资源使用情况: %s\n", strerror(errno));
    }
}

int main() {
    printf("=== 沙箱集成测试 ===\n");
    printf("PID: %d\n", getpid());
    printf("模拟真实的算法竞赛程序，测试沙箱的综合性能\n");
    
    struct timeval total_start, total_end;
    gettimeofday(&total_start, NULL);
    
    // 执行各种测试
    test_fibonacci_large();
    print_resource_usage();
    
    test_shortest_path();
    print_resource_usage();
    
    test_knapsack();
    print_resource_usage();
    
    test_file_io();
    print_resource_usage();
    
    gettimeofday(&total_end, NULL);
    double total_time = (total_end.tv_sec - total_start.tv_sec) + 
                       (total_end.tv_usec - total_start.tv_usec) / 1000000.0;
    
    printf("\n=== 集成测试完成 ===\n");
    printf("总执行时间: %.3f 秒\n", total_time);
    printf("所有测试均在沙箱环境中安全执行\n");
    
    return 0;
}