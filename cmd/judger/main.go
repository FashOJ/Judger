package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/FashOJ/Judger/internal/judge"
	"github.com/FashOJ/Judger/internal/model"
)

func main() {
	log.Println("Starting FashOJ Judger...")

	// 模拟数据
	workDir := "temp"
	_ = os.MkdirAll(workDir, 0755)

	sourceCode := `#include <iostream>
int main() {
    int a, b;
    if (std::cin >> a >> b) {
        std::cout << a + b;
    }
    return 0;
}`

	// 构造任务
	task := &model.JudgeTask{
		ID:          "task-001",
		SourceCode:  sourceCode,
		Language:    model.LangCPP,
		TimeLimit:   1000, // 1000ms
		MemoryLimit: 256,  // 256MB
		WorkDir:     workDir,
		TestCases: []model.TestCase{
			{
				ID:          "case-1",
				Input:       filepath.Join(workDir, "1.in"),
				ExpectedOut: filepath.Join(workDir, "1.out"),
			},
			// 可以添加更多测试用例
		},
	}

	// 运行判题服务
	service := judge.NewJudgeService()
	ctx := context.Background()
	
	result := service.Judge(ctx, task)

	// 输出结果
	outputJSON, _ := json.MarshalIndent(result, "", "  ")
	log.Printf("Judge Result:\n%s\n", string(outputJSON))
}
