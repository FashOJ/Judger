package compiler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/FashOJ/Judger/internal/config"
	"github.com/FashOJ/Judger/internal/model"
)

const (
	CompileTimeLimit = 10 * time.Second // 编译超时时间
)

type Compiler interface {
	Compile(ctx context.Context, sourceCode, workDir string) (string, string, error)
}

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
	// 确保文件对 nobody 用户可读
	_ = os.Chmod(srcPath, 0666)

	// 创建带超时的上下文，如果外部没传超时，这里做一个兜底
	compileCtx, cancel := context.WithTimeout(ctx, CompileTimeLimit)
	defer cancel()

	// g++ main.cpp -o main -O2 -Wall -std=c++17
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

func GetCompiler(lang model.Language) (Compiler, error) {
	switch lang {
	case model.LangCPP:
		return NewCPPCompiler(), nil
	case model.LangPython:
		return NewPythonCompiler(), nil
	case model.LangJava:
		return NewJavaCompiler(), nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}
}

// PythonCompiler Python 编译器（解释器）
// Python 不需要编译，这里主要做语法检查和文件准备
type PythonCompiler struct {
	PythonPath string
}

func NewPythonCompiler() *PythonCompiler {
	// 使用用户指定的 Python 路径
	return &PythonCompiler{
		PythonPath: config.GlobalConfig.Compilers.Python,
	}
}

func (c *PythonCompiler) Compile(ctx context.Context, sourceCode, workDir string) (string, string, error) {
	srcPath := filepath.Join(workDir, "main.py")

	// 对于解释型语言，我们返回的 exePath 实际上是脚本路径
	// 运行的时候需要用 python xxx.py 来执行
	// 为了统一接口，这里我们返回 srcPath，但需要在 Runner 中特殊处理
	// 或者，我们可以生成一个 shell 脚本作为 exePath?
	// 不，更好的方式是在 Runner 中识别语言，或者将 python 路径打包进去。
	// 这里我们约定：如果是 Python，返回的 exePath 是脚本路径。

	if err := os.WriteFile(srcPath, []byte(sourceCode), 0644); err != nil {
		return "", "", fmt.Errorf("failed to write source code: %v", err)
	}

	// 语法检查 (可选)
	// python -m py_compile main.py
	compileCtx, cancel := context.WithTimeout(ctx, CompileTimeLimit)
	defer cancel()

	cmd := exec.CommandContext(compileCtx, c.PythonPath, "-m", "py_compile", srcPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if compileCtx.Err() == context.DeadlineExceeded {
			return "", "Compilation Time Limit Exceeded", fmt.Errorf("compilation timeout")
		}
		// 编译错误（语法错误）
		return "", stderr.String(), fmt.Errorf("compilation failed")
	}

	// 对于 Python，我们返回脚本路径
	// 在 Runner 中，我们需要知道这是 Python 脚本，然后用 Python 解释器去运行它
	// 但 Runner 接口目前只接受 exePath。
	// 我们可以生成一个 wrapper script (chmod +x)
	// #!/bin/sh
	// /path/to/python main.py
	// 这样 Runner 就可以直接 exec 这个 script。

	exePath := filepath.Join(workDir, "run.sh")
	scriptContent := fmt.Sprintf("#!/bin/sh\nexec %s %s", c.PythonPath, srcPath)
	if err := os.WriteFile(exePath, []byte(scriptContent), 0755); err != nil {
		return "", "", fmt.Errorf("failed to create wrapper script: %v", err)
	}

	return exePath, "", nil
}

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
