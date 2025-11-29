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
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewJudgeServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := c.Judge(ctx, &pb.JudgeRequest{
		Id:          "test-001",
		SourceCode:  "#include <iostream>\nint main() { int a, b; std::cin >> a >> b; std::cout << a + b; return 0; }",
		Language:    "cpp",
		TimeLimit:   1000,
		MemoryLimit: 256,
		TestCases: []*pb.TestCase{
			{
				Id:             "case-1",
				Input:          "1 2",
				ExpectedOutput: "3",
			},
		},
	})
	if err != nil {
		log.Fatalf("could not judge: %v", err)
	}
	fmt.Printf("Status: %s, Time: %dms, Memory: %dKB\n", r.Status, r.TimeUsed, r.MemoryUsed)
}
