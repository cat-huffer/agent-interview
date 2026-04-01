// Package router 负责 HTTP 路由的注册和中间件配置。
package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/enterprise/ai-agent-go/internal/handler"
)

// Register 注册所有 HTTP 路由和中间件
func Register(engine *gin.Engine, chatH *handler.ChatHandler, docH *handler.DocumentHandler, healthH *handler.HealthHandler) {
	// 全局中间件
	engine.Use(
		requestIDMiddleware(),
		corsMiddleware(),
		loggerMiddleware(),
	)

	// 健康检查（不受路由组前缀影响）
	engine.GET("/health", healthH.Check)

	// API v1 路由组
	v1 := engine.Group("/api/v1")
	{
		// 对话接口
		v1.POST("/chat", chatH.Chat)
		v1.POST("/chat/stream", chatH.ChatStream)

		// 文档接口
		v1.POST("/documents", docH.Upload)
		v1.GET("/documents/:id", docH.GetStatus)
	}
}

// requestIDMiddleware 为每个请求生成唯一的 Trace ID
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Request-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}
		c.Set("trace_id", traceID)
		c.Header("X-Request-ID", traceID)
		c.Next()
	}
}

// corsMiddleware 跨域资源共享中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// loggerMiddleware 请求日志中间件
func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// Gin 默认 Logger 已输出日志，此处可做额外的结构化日志
		_ = latency
		_ = statusCode
		_ = path
	}
}
