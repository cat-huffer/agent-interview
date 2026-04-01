// Package handler 提供 HTTP 请求处理器。
package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/enterprise/ai-agent-go/internal/agent"
	"github.com/enterprise/ai-agent-go/internal/memory"
	"github.com/enterprise/ai-agent-go/internal/model"
	"github.com/enterprise/ai-agent-go/pkg/common"
)

// ChatHandler 对话请求处理器
type ChatHandler struct {
	orchestrator *agent.Orchestrator
	memManager   *memory.Manager
	logger       *zap.Logger
}

// NewChatHandler 创建对话处理器
func NewChatHandler(orchestrator *agent.Orchestrator, memManager *memory.Manager, logger *zap.Logger) *ChatHandler {
	return &ChatHandler{
		orchestrator: orchestrator,
		memManager:   memManager,
		logger:       logger,
	}
}

// Chat 处理同步对话请求
// POST /api/v1/chat
func (h *ChatHandler) Chat(c *gin.Context) {
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, common.ErrCodeInvalidParam, "请求参数错误: "+err.Error())
		return
	}

	// 如果未提供 SessionID，自动生成
	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
	}

	h.logger.Info("收到对话请求",
		zap.String("session_id", req.SessionID),
		zap.Int("message_len", len(req.Message)),
	)

	ctx := c.Request.Context()

	resp, err := h.orchestrator.ProcessMessage(ctx, &req)
	if err != nil {
		h.logger.Error("对话处理失败", zap.Error(err))
		common.Fail(c, http.StatusInternalServerError, common.ErrInternal(err))
		return
	}

	resp.MessageID = uuid.New().String()
	common.OK(c, resp)
}

// ChatStream 处理 SSE 流式对话请求
// POST /api/v1/chat/stream
func (h *ChatHandler) ChatStream(c *gin.Context) {
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, common.ErrCodeInvalidParam, "请求参数错误: "+err.Error())
		return
	}

	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
	}

	h.logger.Info("收到流式对话请求",
		zap.String("session_id", req.SessionID),
	)

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ctx := c.Request.Context()
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		common.FailWithCode(c, http.StatusInternalServerError, common.ErrCodeInternal, "不支持流式响应")
		return
	}

	// 发送会话 ID
	h.writeSSE(c.Writer, "session", fmt.Sprintf(`{"session_id":"%s"}`, req.SessionID))
	flusher.Flush()

	// 调用编排器处理（此处简化为同步调用后模拟流式输出）
	// 实际生产中应使用真正的流式 LLM 调用
	resp, err := h.orchestrator.ProcessMessage(ctx, &req)
	if err != nil {
		h.writeSSE(c.Writer, "error", fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		flusher.Flush()
		return
	}

	// 模拟流式输出：将完整回答按句子拆分逐步发送
	runes := []rune(resp.Content)
	chunkSize := 20 // 每次发送的字符数
	for i := 0; i < len(runes); i += chunkSize {
		select {
		case <-ctx.Done():
			return
		default:
		}

		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		chunk := model.StreamChunk{
			Event: "message",
			Data:  string(runes[i:end]),
		}
		data, _ := json.Marshal(chunk)
		h.writeSSE(c.Writer, "message", string(data))
		flusher.Flush()

		time.Sleep(50 * time.Millisecond) // 模拟打字效果
	}

	// 发送工具调用信息
	if len(resp.ToolCalls) > 0 {
		toolData, _ := json.Marshal(resp.ToolCalls)
		h.writeSSE(c.Writer, "tool_calls", string(toolData))
		flusher.Flush()
	}

	// 发送引用信息
	if len(resp.References) > 0 {
		refData, _ := json.Marshal(resp.References)
		h.writeSSE(c.Writer, "references", string(refData))
		flusher.Flush()
	}

	// 发送完成事件
	h.writeSSE(c.Writer, "done", `{"status":"completed"}`)
	flusher.Flush()
}

// writeSSE 写入一条 SSE 事件
func (h *ChatHandler) writeSSE(w io.Writer, event, data string) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
}
