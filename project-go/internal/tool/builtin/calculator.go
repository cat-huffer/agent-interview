package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"go.uber.org/zap"

	"github.com/enterprise/ai-agent-go/internal/tool"
)

// CalculatorTool 数学计算工具
type CalculatorTool struct {
	logger *zap.Logger
}

// NewCalculatorTool 创建数学计算工具
func NewCalculatorTool(logger *zap.Logger) *CalculatorTool {
	return &CalculatorTool{logger: logger}
}

// Name 返回工具名称
func (t *CalculatorTool) Name() string { return "calculator" }

// Description 返回工具描述
func (t *CalculatorTool) Description() string {
	return "执行数学计算，支持加减乘除、幂运算和常用数学函数"
}

// Parameters 返回参数 JSON Schema
func (t *CalculatorTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"description": "运算类型：add / subtract / multiply / divide / power / sqrt",
				"enum":        []string{"add", "subtract", "multiply", "divide", "power", "sqrt"},
			},
			"a": map[string]interface{}{
				"type":        "number",
				"description": "第一个操作数",
			},
			"b": map[string]interface{}{
				"type":        "number",
				"description": "第二个操作数（sqrt 运算不需要）",
			},
		},
		"required": []string{"operation", "a"},
	}
}

// Execute 执行数学计算
func (t *CalculatorTool) Execute(ctx context.Context, input string) (*tool.ToolResult, error) {
	var params struct {
		Operation string  `json:"operation"`
		A         float64 `json:"a"`
		B         float64 `json:"b"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return tool.NewErrorResult("参数解析失败: " + err.Error()), nil
	}

	var result float64
	var err error

	switch strings.ToLower(params.Operation) {
	case "add":
		result = params.A + params.B
	case "subtract":
		result = params.A - params.B
	case "multiply":
		result = params.A * params.B
	case "divide":
		if params.B == 0 {
			return tool.NewErrorResult("除数不能为零"), nil
		}
		result = params.A / params.B
	case "power":
		result = math.Pow(params.A, params.B)
	case "sqrt":
		if params.A < 0 {
			return tool.NewErrorResult("不能对负数求平方根"), nil
		}
		result = math.Sqrt(params.A)
	default:
		return tool.NewErrorResult(fmt.Sprintf("不支持的运算类型: %s", params.Operation)), nil
	}

	if err != nil {
		return tool.NewErrorResult(err.Error()), nil
	}

	t.logger.Debug("计算完成",
		zap.String("operation", params.Operation),
		zap.Float64("result", result),
	)

	return tool.NewSuccessResult(fmt.Sprintf("%g", result)), nil
}
