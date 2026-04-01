package model

import "time"

// ChatResponse 对话响应
type ChatResponse struct {
	SessionID  string          `json:"session_id"`
	MessageID  string          `json:"message_id"`
	Content    string          `json:"content"`
	ToolCalls  []ToolCallInfo  `json:"tool_calls,omitempty"`
	References []Reference     `json:"references,omitempty"`
	Usage      *UsageInfo      `json:"usage,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// StreamChunk SSE 流式响应块
type StreamChunk struct {
	Event string `json:"event"` // message / tool_call / done / error
	Data  string `json:"data"`
}

// ToolCallInfo 工具调用信息
type ToolCallInfo struct {
	ToolName string `json:"tool_name"`
	Input    string `json:"input"`
	Output   string `json:"output"`
	Duration int64  `json:"duration_ms"`
}

// Reference RAG 检索引用的文档片段
type Reference struct {
	DocID   string  `json:"doc_id"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// UsageInfo Token 用量统计
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMResponse 大模型返回的响应
type LLMResponse struct {
	Content   string         `json:"content"`
	ToolCalls []LLMToolCall  `json:"tool_calls,omitempty"`
	Usage     *UsageInfo     `json:"usage,omitempty"`
}

// LLMToolCall 大模型发起的工具调用
type LLMToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // function
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// DocumentResponse 文档处理响应
type DocumentResponse struct {
	DocID     string    `json:"doc_id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"` // processing / completed / failed
	ChunkCount int     `json:"chunk_count"`
	CreatedAt time.Time `json:"created_at"`
}
