// Package tool 提供 AI Agent 的工具系统。
// 通过 Go interface 定义统一的工具契约，支持动态注册和路由调度。
package tool

import (
	"context"
)

// Tool 工具接口，所有可被 Agent 调用的工具必须实现此接口
type Tool interface {
	// Name 返回工具的唯一标识名称
	Name() string

	// Description 返回工具的功能描述（会被发送给 LLM 以辅助决策）
	Description() string

	// Parameters 返回工具的参数 JSON Schema 定义
	Parameters() map[string]interface{}

	// Execute 执行工具调用，接受 JSON 格式的参数字符串，返回执行结果
	Execute(ctx context.Context, input string) (*ToolResult, error)
}

// ToolResult 工具执行结果
type ToolResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// NewSuccessResult 创建成功的工具执行结果
func NewSuccessResult(output string) *ToolResult {
	return &ToolResult{
		Success: true,
		Output:  output,
	}
}

// NewErrorResult 创建失败的工具执行结果
func NewErrorResult(err string) *ToolResult {
	return &ToolResult{
		Success: false,
		Error:   err,
	}
}

// Metadata 工具元数据，用于路由决策
type Metadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`    // 工具分类：search / math / database / code
	Tags        []string `json:"tags"`        // 标签，辅助路由
	Timeout     int      `json:"timeout_sec"` // 执行超时（秒）
}
