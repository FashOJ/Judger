/*
 * 测试网络隔离功能
 * 模拟算法竞赛中可能的网络访问尝试
 */
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>
#include <errno.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <sys/types.h>
#include <ifaddrs.h>
#include <net/if.h>

// 测试网络接口
void test_network_interfaces() {
    printf("\n=== 测试网络接口 ===\n");
    
    struct ifaddrs *ifaddrs_ptr, *ifa;
    
    if (getifaddrs(&ifaddrs_ptr) == -1) {
        printf("✗ 无法获取网络接口: %s\n", strerror(errno));
        return;
    }
    
    printf("可用的网络接口:\n");
    for (ifa = ifaddrs_ptr; ifa != NULL; ifa = ifa->ifa_next) {
        if (ifa->ifa_addr == NULL) continue;
        
        printf("  接口: %s\n", ifa->ifa_name);
        printf("    标志: ");
        
        if (ifa->ifa_flags & IFF_UP) printf("UP ");
        if (ifa->ifa_flags & IFF_LOOPBACK) printf("LOOPBACK ");
        if (ifa->ifa_flags & IFF_RUNNING) printf("RUNNING ");
        printf("\n");
        
        // 显示IP地址
        if (ifa->ifa_addr->sa_family == AF_INET) {
            struct sockaddr_in *addr_in = (struct sockaddr_in *)ifa->ifa_addr;
            printf("    IPv4: %s\n", inet_ntoa(addr_in->sin_addr));
        }
    }
    
    freeifaddrs(ifaddrs_ptr);
}

// 测试TCP连接
void test_tcp_connection() {
    printf("\n=== 测试TCP连接 ===\n");
    
    struct {
        const char *host;
        int port;
        const char *description;
    } test_targets[] = {
        {"127.0.0.1", 80, "本地HTTP服务"},
        {"8.8.8.8", 53, "Google DNS"},
        {"baidu.com", 80, "百度HTTP服务"},
        {"github.com", 443, "GitHub HTTPS服务"},
        {"localhost", 22, "本地SSH服务"},
        {NULL, 0, NULL}
    };
    
    for (int i = 0; test_targets[i].host; i++) {
        printf("尝试连接 %s:%d (%s)\n", 
               test_targets[i].host, 
               test_targets[i].port, 
               test_targets[i].description);
        
        int sockfd = socket(AF_INET, SOCK_STREAM, 0);
        if (sockfd < 0) {
            printf("  ✗ 创建socket失败: %s\n", strerror(errno));
            continue;
        }
        
        struct sockaddr_in server_addr;
        memset(&server_addr, 0, sizeof(server_addr));
        server_addr.sin_family = AF_INET;
        server_addr.sin_port = htons(test_targets[i].port);
        
        // 解析主机名
        struct hostent *host_entry = gethostbyname(test_targets[i].host);
        if (host_entry) {
            memcpy(&server_addr.sin_addr, host_entry->h_addr_list[0], host_entry->h_length);
            printf("  ✓ 主机名解析成功: %s\n", inet_ntoa(server_addr.sin_addr));
        } else {
            // 尝试直接解析IP
            if (inet_aton(test_targets[i].host, &server_addr.sin_addr) == 0) {
                printf("  ✗ 主机名解析失败: %s\n", hstrerror(h_errno));
                close(sockfd);
                continue;
            }
        }
        
        // 设置连接超时
        struct timeval timeout;
        timeout.tv_sec = 3;
        timeout.tv_usec = 0;
        setsockopt(sockfd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout));
        setsockopt(sockfd, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout));
        
        // 尝试连接
        if (connect(sockfd, (struct sockaddr*)&server_addr, sizeof(server_addr)) == 0) {
            printf("  ✓ 连接成功\n");
            
            // 发送简单的HTTP请求（如果是HTTP端口）
            if (test_targets[i].port == 80) {
                const char *request = "GET / HTTP/1.0\r\n\r\n";
                if (send(sockfd, request, strlen(request), 0) > 0) {
                    printf("  ✓ 发送HTTP请求成功\n");
                    
                    char buffer[1024];
                    int bytes = recv(sockfd, buffer, sizeof(buffer) - 1, 0);
                    if (bytes > 0) {
                        buffer[bytes] = '\0';
                        printf("  ✓ 收到响应: %d 字节\n", bytes);
                        // 显示响应的前几行
                        char *line = strtok(buffer, "\n");
                        int line_count = 0;
                        while (line && line_count < 3) {
                            printf("    %s\n", line);
                            line = strtok(NULL, "\n");
                            line_count++;
                        }
                    }
                }
            }
        } else {
            printf("  ✗ 连接失败: %s\n", strerror(errno));
        }
        
        close(sockfd);
        printf("\n");
    }
}

// 测试UDP通信
void test_udp_communication() {
    printf("\n=== 测试UDP通信 ===\n");
    
    int sockfd = socket(AF_INET, SOCK_DGRAM, 0);
    if (sockfd < 0) {
        printf("✗ 创建UDP socket失败: %s\n", strerror(errno));
        return;
    }
    
    struct sockaddr_in server_addr;
    memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sin_family = AF_INET;
    server_addr.sin_port = htons(53); // DNS端口
    inet_aton("8.8.8.8", &server_addr.sin_addr);
    
    // 构造简单的DNS查询
    unsigned char dns_query[] = {
        0x12, 0x34, // Transaction ID
        0x01, 0x00, // Flags: standard query
        0x00, 0x01, // Questions: 1
        0x00, 0x00, // Answer RRs: 0
        0x00, 0x00, // Authority RRs: 0
        0x00, 0x00, // Additional RRs: 0
        // Query for "google.com" A record
        0x06, 'g', 'o', 'o', 'g', 'l', 'e',
        0x03, 'c', 'o', 'm',
        0x00, // End of name
        0x00, 0x01, // Type: A
        0x00, 0x01  // Class: IN
    };
    
    printf("尝试发送DNS查询到 8.8.8.8:53\n");
    
    if (sendto(sockfd, dns_query, sizeof(dns_query), 0, 
               (struct sockaddr*)&server_addr, sizeof(server_addr)) > 0) {
        printf("  ✓ DNS查询发送成功\n");
        
        // 尝试接收响应
        unsigned char buffer[512];
        struct sockaddr_in from_addr;
        socklen_t from_len = sizeof(from_addr);
        
        struct timeval timeout;
        timeout.tv_sec = 3;
        timeout.tv_usec = 0;
        setsockopt(sockfd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout));
        
        int bytes = recvfrom(sockfd, buffer, sizeof(buffer), 0, 
                            (struct sockaddr*)&from_addr, &from_len);
        if (bytes > 0) {
            printf("  ✓ 收到DNS响应: %d 字节\n", bytes);
        } else {
            printf("  ✗ 未收到DNS响应: %s\n", strerror(errno));
        }
    } else {
        printf("  ✗ DNS查询发送失败: %s\n", strerror(errno));
    }
    
    close(sockfd);
}

// 测试原始套接字
void test_raw_socket() {
    printf("\n=== 测试原始套接字 ===\n");
    
    int sockfd = socket(AF_INET, SOCK_RAW, IPPROTO_ICMP);
    if (sockfd < 0) {
        printf("✗ 创建原始套接字失败: %s\n", strerror(errno));
        printf("  (这通常需要root权限)\n");
    } else {
        printf("✓ 原始套接字创建成功\n");
        printf("  警告: 原始套接字可能被用于网络攻击\n");
        close(sockfd);
    }
}

int main() {
    printf("=== 网络隔离测试 ===\n");
    printf("PID: %d\n", getpid());
    
    test_network_interfaces();
    test_tcp_connection();
    test_udp_communication();
    test_raw_socket();
    
    printf("\n=== 网络隔离测试完成 ===\n");
    return 0;
}