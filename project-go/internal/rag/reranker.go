package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"go.uber.org/zap"

	"github.com/enterprise/ai-agent-go/internal/llm"
	"github.com/enterprise/ai-agent-go/internal/model"
)

// Reranker 重排序器。
// 使用 LLM 对初检结果进行精排，提高 Top-K 结果的相关性。
type Reranker struct {
	router *llm.Router
	logger *zap.Logger
}

// NewReranker 创建重排序器
func NewReranker(router *llm.Router, logger *zap.Logger) *Reranker {
	return &Reranker{
		router: router,
		logger: logger,
	}
}

const rerankPrompt = `请对以下文档片段与查询的相关性进行评分（0-10分）。

查询：%s

文档片段：
%s

请以 JSON 数组格式返回每个文档的分数：[{"index": 0, "score": 8.5}, ...]`

// Rerank 对检索结果进行重排序。
// 将查询和所有候选文档一起发送给 LLM，由 LLM 评估相关性并重新排序。
func (r *Reranker) Rerank(ctx context.Context, query string, refs []model.Reference) ([]model.Reference, error) {
	if len(refs) <= 1 {
		return refs, nil
	}

	// 构造文档摘要列表
	var docsText string
	for i, ref := range refs {
		content := ref.Content
		if len([]rune(content)) > 200 {
			content = string([]rune(content)[:200]) + "..."
		}
		docsText += fmt.Sprintf("[%d] %s\n", i, content)
	}

	prompt := fmt.Sprintf(rerankPrompt, query, docsText)

	req := &model.LLMRequest{
		Messages: []model.LLMMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
		MaxTokens:   500,
	}

	resp, err := r.router.Chat(ctx, req)
	if err != nil {
		r.logger.Warn("Rerank LLM 调用失败，返回原始排序", zap.Error(err))
		return refs, nil
	}

	// 解析 LLM 返回的评分
	var scores []struct {
		Index int     `json:"index"`
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &scores); err != nil {
		r.logger.Warn("Rerank 结果解析失败，返回原始排序", zap.Error(err))
		return refs, nil
	}

	// 更新分数并重排
	for _, s := range scores {
		if s.Index >= 0 && s.Index < len(refs) {
			refs[s.Index].Score = s.Score
		}
	}

	sort.Slice(refs, func(i, j int) bool {
		return refs[i].Score > refs[j].Score
	})

	r.logger.Info("Rerank 完成", zap.Int("doc_count", len(refs)))
	return refs, nil
}
