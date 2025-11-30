package model

// JudgeStatus 判题状态枚举
type JudgeStatus string

const (
	StatusAccepted            JudgeStatus = "Accepted"
	StatusWrongAnswer         JudgeStatus = "Wrong Answer"
	StatusTimeLimitExceeded   JudgeStatus = "Time Limit Exceeded"
	StatusMemoryLimitExceeded JudgeStatus = "Memory Limit Exceeded"
	StatusPresentationError   JudgeStatus = "Presentation Error"
	StatusRuntimeError        JudgeStatus = "Runtime Error"
	StatusSystemError         JudgeStatus = "System Error"
	StatusUnknownError        JudgeStatus = "Unknown Error"
	StatusCompileError        JudgeStatus = "Compile Error"
	StatusPending             JudgeStatus = "Pending"
)

// Language 编程语言枚举
type Language string

const (
	LangCPP    Language = "cpp"
	LangGo     Language = "go"
	LangJava   Language = "java"
	LangPython Language = "python"
)

// JudgeTask 判题任务
type JudgeTask struct {
	ID          string
	SourceCode  string
	Language    Language
	TimeLimit   int64 // ms
	MemoryLimit int64 // MB
	TestCases   []TestCase
	WorkDir     string
}

// TestCase 测试用例
type TestCase struct {
	ID          string
	Input       string // 输入内容或文件路径
	ExpectedOut string // 预期输出内容或文件路径
}

// JudgeResult 单次判题结果
type JudgeResult struct {
	Status      JudgeStatus
	Message     string
	TimeUsed    int64 // ms
	MemoryUsed  int64 // KB
	CompileLog  string
	CaseResults []CaseResult
}

// CaseResult 单个测试点的结果
type CaseResult struct {
	CaseID      string
	Status      JudgeStatus
	TimeUsed    int64 // ms
	MemoryUsed  int64 // KB
	Message     string
	Input       string
	Output      string
	ExpectedOut string
}
