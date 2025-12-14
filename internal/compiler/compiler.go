package compiler

import (
	"context"
	"fmt"
	"time"

	"github.com/FashOJ/Judger/internal/model"
)

const (
	CompileTimeLimit = 10 * time.Second
)

type Compiler interface {
	Compile(ctx context.Context, sourceCode, workDir string) (string, string, error)
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
