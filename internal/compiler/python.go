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

type PythonCompiler struct {
	PythonPath string
}

func NewPythonCompiler() *PythonCompiler {
	return &PythonCompiler{
		PythonPath: config.GlobalConfig.Compilers.Python,
	}
}

func (c *PythonCompiler) Compile(ctx context.Context, sourceCode, workDir string) (string, string, error) {
	srcPath := filepath.Join(workDir, "main.py")

	if err := os.WriteFile(srcPath, []byte(sourceCode), 0644); err != nil {
		return "", "", fmt.Errorf("failed to write source code: %v", err)
	}

	compileCtx, cancel := context.WithTimeout(ctx, CompileTimeLimit)
	defer cancel()

	cmd := exec.CommandContext(compileCtx, c.PythonPath, "-m", "py_compile", srcPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if compileCtx.Err() == context.DeadlineExceeded {
			return "", "Compilation Time Limit Exceeded", fmt.Errorf("compilation timeout")
		}
		return "", stderr.String(), fmt.Errorf("compilation failed")
	}

	exePath := filepath.Join(workDir, "run.sh")
	scriptContent := fmt.Sprintf("#!/bin/sh\nexec %s %s", c.PythonPath, srcPath)
	if err := os.WriteFile(exePath, []byte(scriptContent), 0755); err != nil {
		return "", "", fmt.Errorf("failed to create wrapper script: %v", err)
	}

	return exePath, "", nil
}

