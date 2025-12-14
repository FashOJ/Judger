package compiler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/FashOJ/Judger/internal/config"
)

type CPPCompiler struct {
	CPPPath string
}

func NewCPPCompiler() *CPPCompiler {
	return &CPPCompiler{
		CPPPath: config.GlobalConfig.Compilers.CPP,
	}
}

func (c *CPPCompiler) Compile(ctx context.Context, sourceCode, workDir string) (string, string, error) {
	srcPath := filepath.Join(workDir, "main.cpp")
	exePath := filepath.Join(workDir, "main")

	if err := os.WriteFile(srcPath, []byte(sourceCode), 0644); err != nil {
		return "", "", fmt.Errorf("failed to write source code: %v", err)
	}
	_ = os.Chmod(srcPath, 0666)

	compileCtx, cancel := context.WithTimeout(ctx, CompileTimeLimit)
	defer cancel()

	cmd := exec.CommandContext(compileCtx, c.CPPPath, srcPath, "-o", exePath, "-O2", "-Wall", "-std=c++17")
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
