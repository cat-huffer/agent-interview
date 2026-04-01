// Package model 定义了 AI Agent 系统中所有的数据传输对象（DTO）和领域实体。
package model

// ChatRequest 对话请求
type ChatRequest struct {
	SessionID string            `json:"session_id" binding:"required"` // 会话 ID
	Message   string            `json:"message" binding:"required"`    // 用户消息
	Stream    bool              `json:"stream"`                        // 是否流式响应
	Options   *ChatOptions      `json:"options,omitempty"`             // 可选参数
	Metadata  map[string]string `json:"metadata,omitempty"`            // 扩展元数据
}

// ChatOptions 对话可选参数
type ChatOptions struct {
	Model       string   `json:"model,omitempty"`        // 指定模型
	Temperature float64  `json:"temperature,omitempty"`  // 温度参数
	MaxTokens   int      `json:"max_tokens,omitempty"`   // 最大 token 数
	Tools       []string `json:"tools,omitempty"`        // 允许使用的工具列表
	EnableRAG   *bool    `json:"enable_rag,omitempty"`   // 是否启用 RAG
}

// DocumentUploadRequest 文档上传请求
type DocumentUploadRequest struct {
	Title       string            `json:"title" binding:"required"`
	Content     string            `json:"content" binding:"required"`
	ContentType string            `json:"content_type"` // text / markdown / html
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// LLMRequest 发送给大模型的请求
type LLMRequest struct {
	Model       string       `json:"model"`
	Messages    []LLMMessage `json:"messages"`
	Temperature float64      `json:"temperature,omitempty"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	Stream      bool         `json:"stream,omitempty"`
	Tools       []ToolDef    `json:"tools,omitempty"`
}

// LLMMessage 大模型消息
type LLMMessage struct {
	Role    string `json:"role"`    // system / user / assistant / tool
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// ToolDef 工具定义（发送给 LLM 的 Function Calling 格式）
type ToolDef struct {
	Type     string       `json:"type"` // function
	Function FunctionDef  `json:"function"`
}

// FunctionDef 函数定义
type FunctionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}
