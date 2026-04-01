package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/enterprise/ai-agent-go/internal/tool"
)

// DatabaseTool 数据库查询工具（只读）
type DatabaseTool struct {
	logger *zap.Logger
}

// NewDatabaseTool 创建数据库查询工具
func NewDatabaseTool(logger *zap.Logger) *DatabaseTool {
	return &DatabaseTool{logger: logger}
}

// Name 返回工具名称
func (t *DatabaseTool) Name() string { return "database_query" }

// Description 返回工具描述
func (t *DatabaseTool) Description() string {
	return "执行只读 SQL 查询，获取数据库中的结构化数据。仅支持 SELECT 语句。"
}

// Parameters 返回参数 JSON Schema
func (t *DatabaseTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sql": map[string]interface{}{
				"type":        "string",
				"description": "SQL 查询语句（仅支持 SELECT）",
			},
			"database": map[string]interface{}{
				"type":        "string",
				"description": "目标数据库名称",
				"default":     "default",
			},
		},
		"required": []string{"sql"},
	}
}

// Execute 执行数据库查询
func (t *DatabaseTool) Execute(ctx context.Context, input string) (*tool.ToolResult, error) {
	var params struct {
		SQL      string `json:"sql"`
		Database string `json:"database"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return tool.NewErrorResult("参数解析失败: " + err.Error()), nil
	}

	// 安全检查：只允许 SELECT 语句
	trimmed := strings.TrimSpace(strings.ToUpper(params.SQL))
	if !strings.HasPrefix(trimmed, "SELECT") {
		return tool.NewErrorResult("安全限制：仅允许 SELECT 查询"), nil
	}

	// 检测危险关键词
	dangerous := []string{"DROP", "DELETE", "UPDATE", "INSERT", "ALTER", "TRUNCATE"}
	for _, kw := range dangerous {
		if strings.Contains(trimmed, kw) {
			return tool.NewErrorResult(fmt.Sprintf("安全限制：SQL 中不允许包含 %s", kw)), nil
		}
	}

	t.logger.Info("执行数据库查询",
		zap.String("sql", params.SQL),
		zap.String("database", params.Database),
	)

	// 实际实现中执行 SQL 查询
	// rows, err := db.QueryContext(ctx, params.SQL)
	result := fmt.Sprintf("查询已执行: %s（数据库连接未配置，此为模拟结果）", params.SQL)

	return tool.NewSuccessResult(result), nil
}
