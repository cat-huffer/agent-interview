package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/enterprise/ai-agent-go/internal/cache"
	"github.com/enterprise/ai-agent-go/internal/vectordb"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	cache    cache.Cache
	vectorDB vectordb.VectorDB
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(cache cache.Cache, vectorDB vectordb.VectorDB) *HealthHandler {
	return &HealthHandler{
		cache:    cache,
		vectorDB: vectorDB,
	}
}

// healthResponse 健康检查响应
type healthResponse struct {
	Status     string            `json:"status"`
	Components map[string]string `json:"components"`
	Timestamp  time.Time         `json:"timestamp"`
}

// Check 执行健康检查
// GET /health
func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	components := make(map[string]string)
	allHealthy := true

	// 检查 Redis
	if h.cache.Healthy(ctx) {
		components["redis"] = "healthy"
	} else {
		components["redis"] = "unhealthy"
		allHealthy = false
	}

	// 检查 Milvus
	if h.vectorDB.Healthy(ctx) {
		components["milvus"] = "healthy"
	} else {
		components["milvus"] = "unhealthy"
		allHealthy = false
	}

	status := "healthy"
	httpCode := http.StatusOK
	if !allHealthy {
		status = "degraded"
		httpCode = http.StatusServiceUnavailable
	}

	c.JSON(httpCode, healthResponse{
		Status:     status,
		Components: components,
		Timestamp:  time.Now(),
	})
}
