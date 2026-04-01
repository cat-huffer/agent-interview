package agent

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/enterprise/ai-agent-go/internal/llm"
	"github.com/enterprise/ai-agent-go/internal/model"
)

const reflectionPrompt = `你是一个回答质量审查专家。请审查以下回答并进行改进。

用户问题：%s

原始回答：%s

请从以下维度审查：
1. 准确性：信息是否准确，有无事实错误
2. 完整性：是否完整回答了用户的问题
3. 清晰度：表达是否清晰，逻辑是否连贯
4. 实用性：是否对用户有实际帮助

如果原始回答已经很好，直接返回原始回答。
如果需要改进，返回改进后的完整回答（不要解释改进了什么，直接给出改进后的答案）。`

// ReflectionAgent 反思型 Agent。
// 对已生成的回答进行自我审查和改进，提升回答质量。
type ReflectionAgent struct {
	router *llm.Router
	logger *zap.Logger
}

// NewReflectionAgent 创建反思 Agent
func NewReflectionAgent(router *llm.Router, logger *zap.Logger) *ReflectionAgent {
	return &ReflectionAgent{
		router: router,
		logger: logger,
	}
}

// Reflect 对初始回答进行反思并改进。
// 返回改进后的回答，如果无需改进则返回空字符串。
func (r *ReflectionAgent) Reflect(ctx context.Context, question, initialAnswer string) (string, error) {
	prompt := fmt.Sprintf(reflectionPrompt, question, initialAnswer)

	req := &model.LLMRequest{
		Messages: []model.LLMMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
	}

	resp, err := r.router.Chat(ctx, req)
	if err != nil {
		r.logger.Warn("反思 Agent 调用失败", zap.Error(err))
		return "", err
	}

	improved := resp.Content

	// 如果改进后的回答和原始回答差异不大，返回空表示无需改进
	if improved == initialAnswer {
		return "", nil
	}

	r.logger.Info("反思改进完成",
		zap.Int("original_len", len(initialAnswer)),
		zap.Int("improved_len", len(improved)),
	)

	return improved, nil
}
