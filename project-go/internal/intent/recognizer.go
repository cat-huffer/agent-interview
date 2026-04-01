// Package intent 提供用户意图识别能力。
// 通过 LLM 分析用户输入，判断其意图类别和置信度，用于 Agent 编排决策。
package intent

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/enterprise/ai-agent-go/internal/llm"
	"github.com/enterprise/ai-agent-go/internal/model"
)

// 意图类型常量
const (
	IntentChat        = "chat"         // 闲聊
	IntentRAGQuery    = "rag_query"    // 知识库查询
	IntentToolUse     = "tool_use"     // 工具调用
	IntentComplexTask = "complex_task" // 复杂任务（需要多步推理）
)

// 意图识别的 Prompt 模板
const intentPrompt = `你是一个意图识别引擎。根据用户输入判断其意图类型。

可选意图：
- chat: 日常闲聊、问候、闲谈
- rag_query: 需要从知识库检索信息回答的问题
- tool_use: 需要调用工具（计算、搜索、数据库查询）才能完成的任务
- complex_task: 需要多步推理、规划的复杂任务

请以 JSON 格式返回：
{"intent": "意图类型", "confidence": 0.0-1.0, "entities": {}, "required_tools": []}

用户输入：%s`

// Recognizer 意图识别器
type Recognizer struct {
	router *llm.Router
	logger *zap.Logger
}

// NewRecognizer 创建意图识别器
func NewRecognizer(router *llm.Router, logger *zap.Logger) *Recognizer {
	return &Recognizer{
		router: router,
		logger: logger,
	}
}

// Recognize 分析用户输入并返回意图识别结果。
// 使用 LLM 进行意图分类，并提取关键实体信息。
func (r *Recognizer) Recognize(ctx context.Context, userInput string) (*model.IntentResult, error) {
	prompt := fmt.Sprintf(intentPrompt, userInput)

	req := &model.LLMRequest{
		Messages: []model.LLMMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1, // 低温度提高确定性
		MaxTokens:   200,
	}

	resp, err := r.router.Chat(ctx, req)
	if err != nil {
		r.logger.Warn("LLM 意图识别失败，降级为 chat", zap.Error(err))
		return r.fallback(), nil
	}

	var result model.IntentResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		r.logger.Warn("解析意图识别结果失败，降级为 chat",
			zap.String("raw", resp.Content),
			zap.Error(err),
		)
		return r.fallback(), nil
	}

	r.logger.Info("意图识别完成",
		zap.String("intent", result.Intent),
		zap.Float64("confidence", result.Confidence),
	)

	return &result, nil
}

// fallback 当 LLM 不可用或解析失败时的兜底策略
func (r *Recognizer) fallback() *model.IntentResult {
	return &model.IntentResult{
		Intent:     IntentChat,
		Confidence: 0.5,
		Entities:   make(map[string]string),
	}
}
