// Package llm 提供大语言模型的客户端抽象、多模型路由和熔断保护。
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/enterprise/ai-agent-go/internal/config"
	"github.com/enterprise/ai-agent-go/internal/model"
)

// Client 大语言模型客户端接口。
// 所有 LLM 提供方（OpenAI、Anthropic、本地模型等）都应实现此接口。
type Client interface {
	// Chat 发送对话请求并返回完整响应
	Chat(ctx context.Context, req *model.LLMRequest) (*model.LLMResponse, error)

	// ChatStream 发送对话请求并以流式方式返回响应。
	// 调用方通过 channel 接收每个响应块，channel 关闭表示流结束。
	ChatStream(ctx context.Context, req *model.LLMRequest) (<-chan StreamEvent, error)

	// Name 返回客户端标识名称
	Name() string

	// Healthy 检查客户端是否可用
	Healthy(ctx context.Context) bool
}

// StreamEvent 流式响应事件
type StreamEvent struct {
	Content string // 文本内容增量
	Done    bool   // 是否结束
	Err     error  // 错误信息
}

// HTTPClient 基于 HTTP 的 LLM 客户端实现（兼容 OpenAI API 格式）
type HTTPClient struct {
	name       string
	provider   string
	apiKey     string
	baseURL    string
	modelName  string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewHTTPClient 创建一个新的 HTTP LLM 客户端
func NewHTTPClient(cfg config.ModelConfig, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		name:      cfg.Name,
		provider:  cfg.Provider,
		apiKey:    cfg.APIKey,
		baseURL:   cfg.BaseURL,
		modelName: cfg.Model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Chat 发送同步对话请求
func (c *HTTPClient) Chat(ctx context.Context, req *model.LLMRequest) (*model.LLMResponse, error) {
	// 使用请求中指定的模型，如果未指定则使用客户端默认模型
	if req.Model == "" {
		req.Model = c.modelName
	}
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM 返回错误状态码 %d: %s", resp.StatusCode, string(respBody))
	}

	// 解析 OpenAI 格式的响应
	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return apiResp.toLLMResponse(), nil
}

// ChatStream 发送流式对话请求
func (c *HTTPClient) ChatStream(ctx context.Context, req *model.LLMRequest) (<-chan StreamEvent, error) {
	if req.Model == "" {
		req.Model = c.modelName
	}
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("LLM 返回错误状态码 %d", resp.StatusCode)
	}

	ch := make(chan StreamEvent, 32)
	go c.readSSEStream(ctx, resp.Body, ch)

	return ch, nil
}

// readSSEStream 在独立的 goroutine 中读取 SSE 流
func (c *HTTPClient) readSSEStream(ctx context.Context, body io.ReadCloser, ch chan<- StreamEvent) {
	defer close(ch)
	defer body.Close()

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			ch <- StreamEvent{Err: ctx.Err()}
			return
		default:
		}

		n, err := body.Read(buf)
		if n > 0 {
			content := string(buf[:n])
			ch <- StreamEvent{Content: content}
		}
		if err != nil {
			if err != io.EOF {
				ch <- StreamEvent{Err: err}
			}
			ch <- StreamEvent{Done: true}
			return
		}
	}
}

// Name 返回客户端名称
func (c *HTTPClient) Name() string {
	return c.name
}

// Healthy 通过简单请求检查模型是否可用
func (c *HTTPClient) Healthy(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req := &model.LLMRequest{
		Model: c.modelName,
		Messages: []model.LLMMessage{
			{Role: "user", Content: "ping"},
		},
		MaxTokens: 5,
	}

	_, err := c.Chat(ctx, req)
	return err == nil
}

// openAIResponse OpenAI 格式的 API 响应
type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content   string           `json:"content"`
			ToolCalls []model.LLMToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Usage *model.UsageInfo `json:"usage,omitempty"`
}

// toLLMResponse 转换为内部统一的 LLM 响应格式
func (r *openAIResponse) toLLMResponse() *model.LLMResponse {
	resp := &model.LLMResponse{
		Usage: r.Usage,
	}
	if len(r.Choices) > 0 {
		resp.Content = r.Choices[0].Message.Content
		resp.ToolCalls = r.Choices[0].Message.ToolCalls
	}
	return resp
}
