package model

import "time"

// Session 会话实体
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Message 消息实体（持久化存储用）
type Message struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Document 文档实体
type Document struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	ContentType string            `json:"content_type"`
	Tags        []string          `json:"tags"`
	Status      string            `json:"status"`
	ChunkCount  int               `json:"chunk_count"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// DocumentChunk 文档分块
type DocumentChunk struct {
	ID         string    `json:"id"`
	DocID      string    `json:"doc_id"`
	Content    string    `json:"content"`
	Embedding  []float32 `json:"embedding"`
	ChunkIndex int       `json:"chunk_index"`
	CreatedAt  time.Time `json:"created_at"`
}

// MemoryEntry 记忆条目
type MemoryEntry struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Summary   string    `json:"summary,omitempty"`
	Embedding []float32 `json:"embedding,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// IntentResult 意图识别结果
type IntentResult struct {
	Intent     string            `json:"intent"`      // chat / rag_query / tool_use / complex_task
	Confidence float64           `json:"confidence"`
	Entities   map[string]string `json:"entities"`
	RequiredTools []string       `json:"required_tools,omitempty"`
}

// AgentStep Agent 执行步骤
type AgentStep struct {
	StepIndex   int       `json:"step_index"`
	Type        string    `json:"type"` // thought / action / observation / reflection
	Content     string    `json:"content"`
	ToolName    string    `json:"tool_name,omitempty"`
	ToolInput   string    `json:"tool_input,omitempty"`
	ToolOutput  string    `json:"tool_output,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}
