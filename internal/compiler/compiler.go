package compiler

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/FashOJ/Judger/internal/model"
)

type Compiler interface {
	Compile(sourceCode, workDir string) (string, error)
}

type CPPCompiler struct{}

func NewCPPCompiler() *CPPCompiler {
	return &CPPCompiler{}
}

func (c *CPPCompiler) Compile(sourceCode, workDir string) (string, error) {
	srcPath := filepath.Join(workDir, "main.cpp")
	exePath := filepath.Join(workDir, "main")

	if err := os.WriteFile(srcPath, []byte(sourceCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write source code: %v", err)
	}

	// g++ main.cpp -o main -O2 -Wall -std=c++17
	cmd := exec.Command("g++", srcPath, "-o", exePath, "-O2", "-Wall", "-std=c++17")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("compilation failed: %s", stderr.String())
	}

	return exePath, nil
}

func GetCompiler(lang model.Language) (Compiler, error) {
	switch lang {
	case model.LangCPP:
		return NewCPPCompiler(), nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}
}
