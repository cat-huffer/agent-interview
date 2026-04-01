package llm

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"go.uber.org/zap"

	"github.com/enterprise/ai-agent-go/internal/config"
	"github.com/enterprise/ai-agent-go/internal/model"
	"github.com/enterprise/ai-agent-go/pkg/common"
)

// Router 模型路由器。
// 根据请求参数和模型健康状态智能选择最合适的 LLM 客户端，
// 并通过熔断器保护每个模型的调用链路。
type Router struct {
	mu       sync.RWMutex
	clients  map[string]Client          // 模型名称 -> 客户端
	breakers map[string]*CircuitBreaker // 模型名称 -> 熔断器
	default_ string                     // 默认模型名称
	logger   *zap.Logger
}

// NewRouter 创建模型路由器。
// 为每个注册的客户端自动创建独立的熔断器实例。
func NewRouter(clients map[string]Client, defaultModel string, cbCfg config.CBConfig) *Router {
	breakers := make(map[string]*CircuitBreaker, len(clients))
	for name := range clients {
		cb := NewCircuitBreaker(cbCfg.FailureThreshold, cbCfg.SuccessThreshold, cbCfg.Timeout)
		breakers[name] = cb
	}

	return &Router{
		clients:  clients,
		breakers: breakers,
		default_: defaultModel,
	}
}

// SetLogger 设置日志记录器
func (r *Router) SetLogger(logger *zap.Logger) {
	r.logger = logger
}

// Chat 通过路由选择模型并发送对话请求。
// 如果请求指定了模型则使用指定模型，否则使用默认模型。
// 当目标模型熔断时自动降级到其他可用模型。
func (r *Router) Chat(ctx context.Context, req *model.LLMRequest) (*model.LLMResponse, error) {
	client, err := r.selectClient(req.Model)
	if err != nil {
		return nil, err
	}

	modelName := client.Name()
	breaker := r.getBreaker(modelName)

	if !breaker.Allow() {
		// 主模型熔断，尝试降级
		fallback, fbErr := r.findFallback(modelName)
		if fbErr != nil {
			return nil, common.ErrCircuitOpen(modelName)
		}
		r.log("模型 %s 熔断，降级到 %s", modelName, fallback.Name())
		client = fallback
		breaker = r.getBreaker(fallback.Name())
	}

	resp, err := client.Chat(ctx, req)
	if err != nil {
		breaker.RecordFailure()
		return nil, common.WrapError(common.ErrCodeLLMFailed, "LLM 调用失败", err)
	}

	breaker.RecordSuccess()
	return resp, nil
}

// ChatStream 通过路由选择模型并发送流式对话请求
func (r *Router) ChatStream(ctx context.Context, req *model.LLMRequest) (<-chan StreamEvent, error) {
	client, err := r.selectClient(req.Model)
	if err != nil {
		return nil, err
	}

	modelName := client.Name()
	breaker := r.getBreaker(modelName)

	if !breaker.Allow() {
		fallback, fbErr := r.findFallback(modelName)
		if fbErr != nil {
			return nil, common.ErrCircuitOpen(modelName)
		}
		client = fallback
	}

	ch, err := client.ChatStream(ctx, req)
	if err != nil {
		breaker.RecordFailure()
		return nil, common.WrapError(common.ErrCodeLLMFailed, "LLM 流式调用失败", err)
	}

	breaker.RecordSuccess()
	return ch, nil
}

// ListModels 返回所有已注册模型的名称和健康状态
func (r *Router) ListModels() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]string, len(r.clients))
	for name := range r.clients {
		if breaker, ok := r.breakers[name]; ok {
			result[name] = breaker.State().String()
		} else {
			result[name] = "UNKNOWN"
		}
	}
	return result
}

// selectClient 根据模型名称选择客户端
func (r *Router) selectClient(modelName string) (Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	name := modelName
	if name == "" {
		name = r.default_
	}

	client, ok := r.clients[name]
	if !ok {
		return nil, fmt.Errorf("未注册的模型: %s", name)
	}
	return client, nil
}

// findFallback 在主模型不可用时寻找备选模型。
// 按模型名称排序选择第一个未熔断的模型。
func (r *Router) findFallback(excludeModel string) (Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 收集可用模型并排序，保证降级行为可预测
	var candidates []string
	for name := range r.clients {
		if name == excludeModel {
			continue
		}
		candidates = append(candidates, name)
	}
	sort.Strings(candidates)

	for _, name := range candidates {
		if breaker, ok := r.breakers[name]; ok && breaker.Allow() {
			return r.clients[name], nil
		}
	}

	return nil, fmt.Errorf("所有模型均不可用")
}

// getBreaker 获取指定模型的熔断器
func (r *Router) getBreaker(name string) *CircuitBreaker {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if b, ok := r.breakers[name]; ok {
		return b
	}
	// 兜底：返回一个始终允许的默认熔断器
	return NewCircuitBreaker(100, 1, 0)
}

// log 输出日志（如果 logger 已设置）
func (r *Router) log(format string, args ...interface{}) {
	if r.logger != nil {
		r.logger.Info(fmt.Sprintf(format, args...))
	}
}
