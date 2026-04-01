package tool

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/enterprise/ai-agent-go/pkg/common"
)

// Router 工具路由器。
// 根据工具名称从注册中心获取工具并执行，提供超时控制和错误处理。
type Router struct {
	registry *Registry
	logger   *zap.Logger
	timeout  time.Duration // 默认工具执行超时
}

// NewRouter 创建工具路由器
func NewRouter(registry *Registry, logger *zap.Logger) *Router {
	return &Router{
		registry: registry,
		logger:   logger,
		timeout:  30 * time.Second,
	}
}

// Execute 根据工具名称路由并执行工具调用。
// 自动添加超时控制，记录执行耗时。
func (r *Router) Execute(ctx context.Context, toolName, input string) (*ToolResult, error) {
	t, ok := r.registry.Get(toolName)
	if !ok {
		return nil, common.ErrToolNotFound(toolName)
	}

	// 使用带超时的子 Context
	execCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	start := time.Now()
	r.logger.Info("开始执行工具",
		zap.String("tool", toolName),
		zap.String("input", truncate(input, 200)),
	)

	result, err := t.Execute(execCtx, input)
	elapsed := time.Since(start)

	if err != nil {
		r.logger.Error("工具执行失败",
			zap.String("tool", toolName),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
		return nil, common.WrapError(common.ErrCodeToolFailed,
			fmt.Sprintf("工具 %s 执行失败", toolName), err)
	}

	r.logger.Info("工具执行完成",
		zap.String("tool", toolName),
		zap.Duration("elapsed", elapsed),
		zap.Bool("success", result.Success),
	)

	return result, nil
}

// BatchExecute 并发执行多个工具调用。
// 使用 goroutine 并发执行，通过 channel 收集结果。
func (r *Router) BatchExecute(ctx context.Context, calls []ToolCall) []*ToolCallResult {
	results := make([]*ToolCallResult, len(calls))
	ch := make(chan indexedResult, len(calls))

	for i, call := range calls {
		go func(idx int, c ToolCall) {
			result, err := r.Execute(ctx, c.Name, c.Input)
			ch <- indexedResult{Index: idx, Result: result, Err: err}
		}(i, call)
	}

	for range calls {
		ir := <-ch
		results[ir.Index] = &ToolCallResult{
			ToolName: calls[ir.Index].Name,
			Result:   ir.Result,
			Err:      ir.Err,
		}
	}

	return results
}

// ListAvailableTools 返回注册中心中所有可用工具的名称
func (r *Router) ListAvailableTools() []string {
	return r.registry.List()
}

// ToolCall 工具调用请求
type ToolCall struct {
	Name  string
	Input string
}

// ToolCallResult 工具调用结果（含工具名）
type ToolCallResult struct {
	ToolName string
	Result   *ToolResult
	Err      error
}

type indexedResult struct {
	Index  int
	Result *ToolResult
	Err    error
}

// truncate 截断字符串，超过 maxLen 的部分用省略号代替
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
