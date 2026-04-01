package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/enterprise/ai-agent-go/internal/model"
	"github.com/enterprise/ai-agent-go/pkg/common"
)

// DocumentHandler 文档处理器
type DocumentHandler struct {
	logger *zap.Logger
}

// NewDocumentHandler 创建文档处理器
func NewDocumentHandler(logger *zap.Logger) *DocumentHandler {
	return &DocumentHandler{
		logger: logger,
	}
}

// Upload 上传文档
// POST /api/v1/documents
func (h *DocumentHandler) Upload(c *gin.Context) {
	var req model.DocumentUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, common.ErrCodeInvalidParam, "请求参数错误: "+err.Error())
		return
	}

	docID := uuid.New().String()

	h.logger.Info("收到文档上传请求",
		zap.String("doc_id", docID),
		zap.String("title", req.Title),
		zap.Int("content_len", len(req.Content)),
	)

	// 实际项目中此处应异步提交到 ETL 流水线处理
	// pipeline.ProcessDocument(ctx, doc)

	resp := &model.DocumentResponse{
		DocID:      docID,
		Title:      req.Title,
		Status:     "processing",
		CreatedAt:  time.Now(),
	}

	common.OK(c, resp)
}

// GetStatus 查询文档处理状态
// GET /api/v1/documents/:id
func (h *DocumentHandler) GetStatus(c *gin.Context) {
	docID := c.Param("id")
	if docID == "" {
		common.FailWithCode(c, http.StatusBadRequest, common.ErrCodeInvalidParam, "文档 ID 不能为空")
		return
	}

	// 实际项目中从数据库查询文档状态
	// doc, err := docRepo.FindByID(ctx, docID)

	resp := &model.DocumentResponse{
		DocID:      docID,
		Title:      "示例文档",
		Status:     "completed",
		ChunkCount: 10,
		CreatedAt:  time.Now(),
	}

	common.OK(c, resp)
}
