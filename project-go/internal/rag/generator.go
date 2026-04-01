package rag

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/enterprise/ai-agent-go/internal/llm"
	"github.com/enterprise/ai-agent-go/internal/model"
)

// Generator RAG 答案生成器。
// 将检索到的文档片段和用户问题组合成 Prompt，由 LLM 生成最终答案。
type Generator struct {
	router *llm.Router
	logger *zap.Logger
}

// NewGenerator 创建答案生成器
func NewGenerator(router *llm.Router, logger *zap.Logger) *Generator {
	return &Generator{
		router: router,
		logger: logger,
	}
}

const ragSystemPrompt = `你是一个智能知识助手。请根据以下参考文档回答用户的问题。

规则：
1. 仅基于提供的参考文档回答，不要编造信息
2. 如果参考文档中没有相关信息，请明确告知用户
3. 在回答中引用来源文档编号，格式为 [1]、[2] 等
4. 回答要准确、简洁、有条理`

// Generate 基于检索到的文档和用户问题生成答案
func (g *Generator) Generate(ctx context.Context, query string, refs []model.Reference) (string, error) {
	if len(refs) == 0 {
		return g.generateWithoutRefs(ctx, query)
	}

	// 构造参考文档上下文
	var contextParts []string
	for i, ref := range refs {
		contextParts = append(contextParts,
			fmt.Sprintf("[%d] (来源: %s, 相关度: %.2f)\n%s", i+1, ref.DocID, ref.Score, ref.Content),
		)
	}
	contextText := strings.Join(contextParts, "\n\n")

	userPrompt := fmt.Sprintf("参考文档：\n%s\n\n用户问题：%s", contextText, query)

	req := &model.LLMRequest{
		Messages: []model.LLMMessage{
			{Role: "system", Content: ragSystemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	resp, err := g.router.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("RAG 生成答案失败: %w", err)
	}

	g.logger.Info("RAG 答案生成完成",
		zap.Int("ref_count", len(refs)),
		zap.Int("answer_len", len(resp.Content)),
	)

	return resp.Content, nil
}

// generateWithoutRefs 在没有检索到相关文档时直接生成回答
func (g *Generator) generateWithoutRefs(ctx context.Context, query string) (string, error) {
	req := &model.LLMRequest{
		Messages: []model.LLMMessage{
			{
				Role:    "system",
				Content: "你是一个智能助手。知识库中没有找到相关信息，请根据你的知识尽可能回答用户问题，并说明回答未基于内部文档。",
			},
			{Role: "user", Content: query},
		},
	}

	resp, err := g.router.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("生成答案失败: %w", err)
	}

	return resp.Content, nil
}
