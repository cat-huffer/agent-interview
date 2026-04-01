package memory

import (
	"context"

	"github.com/enterprise/ai-agent-go/internal/model"
	"github.com/enterprise/ai-agent-go/internal/vectordb"
)

const longTermCollection = "long_term_memory"

// LongTermMemory 基于向量数据库的长期记忆实现。
// 将对话内容向量化后存入 Milvus，支持按语义相似度检索历史记忆。
type LongTermMemory struct {
	vectorDB vectordb.VectorDB
}

// NewLongTermMemory 创建长期记忆实例
func NewLongTermMemory(vectorDB vectordb.VectorDB) *LongTermMemory {
	return &LongTermMemory{vectorDB: vectorDB}
}

// Save 将记忆条目向量化并存入向量数据库
func (l *LongTermMemory) Save(ctx context.Context, entry *model.MemoryEntry) error {
	// 实际项目中需要先调用 Embedding 模型生成向量
	// embedding, err := embeddingClient.Embed(ctx, entry.Content)
	// 此处使用占位向量
	record := vectordb.VectorRecord{
		ID:      entry.ID,
		Content: entry.Content,
		Embedding: entry.Embedding,
		Metadata: map[string]string{
			"session_id": entry.SessionID,
			"role":       entry.Role,
		},
	}

	return l.vectorDB.Insert(ctx, longTermCollection, []vectordb.VectorRecord{record})
}

// Load 长期记忆不支持按会话顺序加载，返回空结果。
// 如需按时间顺序加载历史记录，应使用关系数据库。
func (l *LongTermMemory) Load(_ context.Context, _ string, _ int) ([]*model.MemoryEntry, error) {
	return nil, nil
}

// Search 通过向量相似度检索与查询语义相关的历史记忆
func (l *LongTermMemory) Search(ctx context.Context, query string, topK int) ([]*model.MemoryEntry, error) {
	// 实际项目中需要先将 query 向量化
	// queryVector, err := embeddingClient.Embed(ctx, query)
	var queryVector []float32 // 占位

	results, err := l.vectorDB.Search(ctx, longTermCollection, queryVector, topK)
	if err != nil {
		return nil, err
	}

	entries := make([]*model.MemoryEntry, 0, len(results))
	for _, r := range results {
		entries = append(entries, &model.MemoryEntry{
			ID:        r.ID,
			Content:   r.Content,
			SessionID: r.Metadata["session_id"],
			Role:      r.Metadata["role"],
		})
	}

	return entries, nil
}

// Clear 从向量数据库中清除指定会话的所有记忆
func (l *LongTermMemory) Clear(ctx context.Context, sessionID string) error {
	// 实际实现中需要通过 sessionID 过滤后删除
	// 向量数据库通常不支持属性过滤删除，可能需要先查再删
	_ = ctx
	_ = sessionID
	return nil
}
