package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Config 配置结构
type Config struct {
	WorkDir string
}

// CompilationResult 编译结果
type CompilationResult struct {
	Success bool
	Message string
	ExePath string
}

// JudgeResult 判题结果
type JudgeResult struct {
	Status   string // "AC", "WA", "RE", "TLE"
	Message  string
	TimeUsed int64 // ms
	MemUsed  int64 // kb
}

func main() {
	log.Println("Starting FashOJ Judger (Lite Version)...")

	// 1. 确定工作目录和文件路径
	workDir := "temp"
	srcPath := filepath.Join(workDir, "main.cpp")
	inputPath := filepath.Join(workDir, "1.in")
	expectedOutputPath := filepath.Join(workDir, "1.out")

	// 2. 编译代码
	log.Printf("Compiling %s...\n", srcPath)
	compResult := compile(srcPath, workDir)
	if !compResult.Success {
		log.Fatalf("Compilation Failed:\n%s", compResult.Message)
	}
	log.Println("Compilation Successful!")

	// 3. 运行程序并获取输出
	log.Printf("Running %s with input %s...\n", compResult.ExePath, inputPath)
	actualOutput, err := run(compResult.ExePath, inputPath)
	if err != nil {
		log.Fatalf("Runtime Error: %v", err)
	}

	// 4. 比较结果
	log.Println("Comparing output...")
	result := compare(actualOutput, expectedOutputPath)

	// 5. 输出判题结果
	log.Println("========================================")
	log.Printf("Judge Result: %s\n", result.Status)
	log.Printf("Message: %s\n", result.Message)
	log.Println("========================================")
}

func compile(srcPath, outDir string) CompilationResult {
	exePath := filepath.Join(outDir, "main")
	// 使用 g++ 编译
	cmd := exec.Command("g++", srcPath, "-o", exePath, "-O2", "-Wall")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return CompilationResult{
			Success: false,
			Message: stderr.String(),
		}
	}

	return CompilationResult{
		Success: true,
		ExePath: exePath,
	}
}

func run(exePath, inputPath string) (string, error) {
	cmd := exec.Command(exePath)

	// 打开输入文件
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to open input file: %v", err)
	}
	defer inputFile.Close()
	cmd.Stdin = inputFile

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("execution failed: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

func compare(actualOutput string, expectedPath string) JudgeResult {
	// 读取预期输出文件
	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		return JudgeResult{
			Status:  "System Error",
			Message: fmt.Sprintf("Failed to read expected output: %v", err),
		}
	}

	// 简单的字符串比较（去除首尾空白字符）
	actual := strings.TrimSpace(actualOutput)
	expected := strings.TrimSpace(string(expectedBytes))

	if actual == expected {
		return JudgeResult{
			Status:  "Accepted",
			Message: "Output matches expected result.",
		}
	} else {
		return JudgeResult{
			Status:  "Wrong Answer",
			Message: fmt.Sprintf("Expected: '%s', Got: '%s'", expected, actual),
		}
	}
}
