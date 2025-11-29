package compiler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/FashOJ/Judger/internal/model"
)

const (
	CompileTimeLimit = 10 * time.Second // 编译超时时间
)

type Compiler interface {
	Compile(ctx context.Context, sourceCode, workDir string) (string, string, error)
}

type CPPCompiler struct{}

func NewCPPCompiler() *CPPCompiler {
	return &CPPCompiler{}
}

func (c *CPPCompiler) Compile(ctx context.Context, sourceCode, workDir string) (string, string, error) {
	srcPath := filepath.Join(workDir, "main.cpp")
	exePath := filepath.Join(workDir, "main")

	if err := os.WriteFile(srcPath, []byte(sourceCode), 0644); err != nil {
		return "", "", fmt.Errorf("failed to write source code: %v", err)
	}

	// 创建带超时的上下文，如果外部没传超时，这里做一个兜底
	compileCtx, cancel := context.WithTimeout(ctx, CompileTimeLimit)
	defer cancel()

	// g++ main.cpp -o main -O2 -Wall -std=c++17
	cmd := exec.CommandContext(compileCtx, "g++", srcPath, "-o", exePath, "-O2", "-Wall", "-std=c++17")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if compileCtx.Err() == context.DeadlineExceeded {
			return "", "Compilation Time Limit Exceeded", fmt.Errorf("compilation timeout")
		}
		return "", stderr.String(), fmt.Errorf("compilation failed")
	}

	return exePath, stderr.String(), nil
}

func GetCompiler(lang model.Language) (Compiler, error) {
	switch lang {
	case model.LangCPP:
		return NewCPPCompiler(), nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}
}
