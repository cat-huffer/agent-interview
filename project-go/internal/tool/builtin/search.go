// Package builtin 提供 AI Agent 的内置工具实现。
package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/enterprise/ai-agent-go/internal/tool"
)

// SearchTool 网络搜索工具
type SearchTool struct {
	logger *zap.Logger
}

// NewSearchTool 创建网络搜索工具
func NewSearchTool(logger *zap.Logger) *SearchTool {
	return &SearchTool{logger: logger}
}

// Name 返回工具名称
func (t *SearchTool) Name() string { return "web_search" }

// Description 返回工具描述
func (t *SearchTool) Description() string {
	return "搜索互联网获取最新信息，适用于需要实时数据或外部知识的查询"
}

// Parameters 返回参数 JSON Schema
func (t *SearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "搜索关键词",
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "最大返回结果数",
				"default":     5,
			},
		},
		"required": []string{"query"},
	}
}

// Execute 执行搜索
func (t *SearchTool) Execute(ctx context.Context, input string) (*tool.ToolResult, error) {
	var params struct {
		Query      string `json:"query"`
		MaxResults int    `json:"max_results"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return tool.NewErrorResult("参数解析失败: " + err.Error()), nil
	}

	if params.Query == "" {
		return tool.NewErrorResult("搜索关键词不能为空"), nil
	}

	if params.MaxResults <= 0 {
		params.MaxResults = 5
	}

	t.logger.Info("执行网络搜索", zap.String("query", params.Query))

	// 实际实现中调用搜索 API（Google/Bing/SerpAPI）
	// results, err := searchAPI.Search(ctx, params.Query, params.MaxResults)
	result := fmt.Sprintf("搜索「%s」返回 %d 条结果（搜索 API 未配置，此为模拟结果）",
		params.Query, params.MaxResults)

	return tool.NewSuccessResult(result), nil
}
