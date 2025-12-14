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

type JavaCompiler struct {
	JavacPath string
}

func NewJavaCompiler() *JavaCompiler {
	return &JavaCompiler{
		JavacPath: config.GlobalConfig.Compilers.Java,
	}
}

func (c *JavaCompiler) Compile(ctx context.Context, sourceCode, workDir string) (string, string, error) {
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve workDir: %v", err)
	}

	srcPath := filepath.Join(absWorkDir, "Main.java")
	if err := os.WriteFile(srcPath, []byte(sourceCode), 0644); err != nil {
		return "", "", fmt.Errorf("failed to write source code: %v", err)
	}

	compileCtx, cancel := context.WithTimeout(ctx, CompileTimeLimit)
	defer cancel()

	cmd := exec.CommandContext(compileCtx, c.JavacPath, "-encoding", "UTF-8", "Main.java")
	cmd.Dir = absWorkDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if compileCtx.Err() == context.DeadlineExceeded {
			return "", "Compilation Time Limit Exceeded", fmt.Errorf("compilation timeout")
		}
		return "", stderr.String(), fmt.Errorf("compilation failed")
	}

	exePath := filepath.Join(absWorkDir, "run_java.sh")
	scriptContent := fmt.Sprintf("#!/bin/sh\nexec /usr/bin/java -cp \"%s\" Main\n", absWorkDir)
	if err := os.WriteFile(exePath, []byte(scriptContent), 0755); err != nil {
		return "", "", fmt.Errorf("failed to create wrapper script: %v", err)
	}

	return exePath, "", nil
}
