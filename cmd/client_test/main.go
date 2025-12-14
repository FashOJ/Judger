package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/FashOJ/Judger/api/proto/judger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewJudgeServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Test AC
	test(ctx, c, "AC Test",
		"#include <iostream>\nint main() { int a, b; std::cin >> a >> b; std::cout << a + b; return 0; }",
		"1 2", "3")

	// 2. Test WA
	test(ctx, c, "WA Test",
		"#include <iostream>\nint main() { int a, b; std::cin >> a >> b; std::cout << a + b + 1; return 0; }",
		"1 2", "3")

	// 3. Test PE (Output: "3 " vs Expected: "3") - Note: Our current AC logic is loose (TrimSpace), so this might AC
	// Let's try something that is definitely PE in strict mode but we need to ensure our PE logic works.
	// Current logic: AC = TrimSpace equal. PE = RemoveAllWhitespace equal.
	// So "3 " vs "3" is AC.
	// Let's try "3\n" vs "3" -> AC.
	// Let's try "1 2" vs "1\n2".
	// TrimSpace("1 2") != TrimSpace("1\n2"). RemoveAllWhitespace("1 2") == "12" == RemoveAllWhitespace("1\n2"). -> PE
	test(ctx, c, "PE Test",
		"#include <iostream>\nint main() { std::cout << \"1 2\"; return 0; }",
		"", "1\n2")

	// 4. Test OLE (Now should be TLE or MLE)
	test(ctx, c, "OLE Test",
		"#include <iostream>\nint main() { while(1) std::cout << \"output limit exceeded...\"; return 0; }",
		"", "expected")

	// 5. Test MLE
	test(ctx, c, "MLE Test",
		"#include <iostream>\n#include <vector>\nint main() { std::vector<int> v; while(1) v.push_back(1); return 0; }",
		"", "expected")

	// 6. Test RE (Div by zero)
	test(ctx, c, "RE Test",
		"#include <iostream>\nint main() { int a = 0; std::cout << 1/a; return 0; }",
		"", "expected")

	// 7. Test Python AC
	testLang(ctx, c, "Python_AC_Test",
		"a, b = map(int, input().split())\nprint(a + b)",
		"python",
		"1 2", "3")

	// 8. Test Java AC
	testLang(ctx, c, "Java_AC_Test",
		"import java.util.*;\npublic class Main {\n  public static void main(String[] args) {\n    Scanner sc = new Scanner(System.in);\n    int a = sc.nextInt();\n    int b = sc.nextInt();\n    System.out.print(a + b);\n  }\n}\n",
		"java",
		"1 2", "3")
}

func test(ctx context.Context, c pb.JudgeServiceClient, name, code, input, expected string) {
	testLang(ctx, c, name, code, "cpp", input, expected)
}

func testLang(ctx context.Context, c pb.JudgeServiceClient, name, code, lang, input, expected string) {
	fmt.Printf("Running %s...\n", name)

	// 根据语言调整默认限制
	timeLimit := int64(1000)
	memoryLimit := int64(128)
	if lang == "python" || lang == "java" {
		timeLimit = 3000  // 3s
		memoryLimit = 512 // 512MB
	}

	r, err := c.Judge(ctx, &pb.JudgeRequest{
		Id:          "test-" + name,
		SourceCode:  code,
		Language:    lang,
		TimeLimit:   timeLimit,
		MemoryLimit: memoryLimit,
		TestCases: []*pb.TestCase{
			{
				Id:             "case-1",
				Input:          input,
				ExpectedOutput: expected,
			},
		},
	})
	if err != nil {
		log.Printf("RPC failed: %v\n", err)
		return
	}
	fmt.Printf("[%s] Status: %s, Message: %s\n", name, r.Status, r.Message)
	if len(r.CaseResults) > 0 {
		fmt.Printf("Case Status: %s\n", r.CaseResults[0].Status)
	}
	fmt.Println("--------------------------------------------------")
}
