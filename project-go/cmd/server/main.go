// Package main 是 AI Agent 服务的程序入口。
// 负责加载配置、初始化依赖、注册路由并启动 HTTP 服务器，
// 同时支持优雅关停（Graceful Shutdown）。
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/enterprise/ai-agent-go/internal/agent"
	"github.com/enterprise/ai-agent-go/internal/cache"
	"github.com/enterprise/ai-agent-go/internal/config"
	"github.com/enterprise/ai-agent-go/internal/handler"
	"github.com/enterprise/ai-agent-go/internal/intent"
	"github.com/enterprise/ai-agent-go/internal/llm"
	"github.com/enterprise/ai-agent-go/internal/memory"
	"github.com/enterprise/ai-agent-go/internal/rag"
	"github.com/enterprise/ai-agent-go/internal/router"
	"github.com/enterprise/ai-agent-go/internal/tool"
	toolbuiltin "github.com/enterprise/ai-agent-go/internal/tool/builtin"
	"github.com/enterprise/ai-agent-go/internal/trace"
	"github.com/enterprise/ai-agent-go/internal/vectordb"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	// ======================== 1. 加载配置 ========================
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// ======================== 2. 初始化日志 ========================
	logger := initLogger(cfg.Log)
	defer logger.Sync()
	logger.Info("AI Agent 服务启动中",
		zap.String("version", version),
		zap.String("build_time", buildTime),
	)

	// ======================== 3. 初始化基础设施 ========================
	// Redis 缓存
	redisCache, err := cache.NewRedisCache(cfg.Redis)
	if err != nil {
		logger.Fatal("初始化 Redis 失败", zap.Error(err))
	}
	defer redisCache.Close()

	// Milvus 向量数据库
	milvusClient, err := vectordb.NewMilvusClient(cfg.Milvus)
	if err != nil {
		logger.Fatal("初始化 Milvus 失败", zap.Error(err))
	}
	defer milvusClient.Close()

	// 链路追踪
	tp, err := trace.InitTracer("ai-agent-go")
	if err != nil {
		logger.Warn("初始化链路追踪失败，将降级运行", zap.Error(err))
	}
	if tp != nil {
		defer tp.Shutdown(context.Background())
	}

	// ======================== 4. 初始化核心组件 ========================
	// LLM 客户端和模型路由
	llmClients := make(map[string]llm.Client)
	for _, modelCfg := range cfg.LLM.Models {
		client := llm.NewHTTPClient(modelCfg, cfg.LLM.RequestTimeout)
		llmClients[modelCfg.Name] = client
	}
	modelRouter := llm.NewRouter(llmClients, cfg.LLM.DefaultModel, cfg.LLM.CircuitBreaker)

	// 记忆管理器
	shortTermMem := memory.NewShortTermMemory(redisCache, 20)
	longTermMem := memory.NewLongTermMemory(milvusClient)
	memManager := memory.NewManager(shortTermMem, longTermMem)

	// 工具系统
	toolRegistry := tool.NewRegistry()
	registerBuiltinTools(toolRegistry, logger)
	toolRouter := tool.NewRouter(toolRegistry, logger)

	// 意图识别器
	intentRecognizer := intent.NewRecognizer(modelRouter, logger)

	// RAG 引擎
	retriever := rag.NewRetriever(milvusClient, redisCache, logger)
	reranker := rag.NewReranker(modelRouter, logger)
	generator := rag.NewGenerator(modelRouter, logger)

	// ======================== 5. 初始化 Agent 编排器 ========================
	orchestrator := agent.NewOrchestrator(agent.OrchestratorDeps{
		ModelRouter:      modelRouter,
		MemoryManager:    memManager,
		ToolRouter:       toolRouter,
		IntentRecognizer: intentRecognizer,
		Retriever:        retriever,
		Reranker:         reranker,
		Generator:        generator,
		Config:           cfg.Agent,
		Logger:           logger,
	})

	// ======================== 6. 初始化 HTTP 处理器 ========================
	chatHandler := handler.NewChatHandler(orchestrator, memManager, logger)
	docHandler := handler.NewDocumentHandler(logger)
	healthHandler := handler.NewHealthHandler(redisCache, milvusClient)

	// ======================== 7. 配置路由并启动服务器 ========================
	gin.SetMode(cfg.Server.Mode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	router.Register(engine, chatHandler, docHandler, healthHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 在 goroutine 中启动服务器
	go func() {
		logger.Info("HTTP 服务器已启动", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP 服务器异常退出", zap.Error(err))
		}
	}()

	// ======================== 8. 优雅关停 ========================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("收到关停信号，开始优雅关停...", zap.String("signal", sig.String()))

	// 给予 15 秒的关停超时
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("HTTP 服务器关停失败", zap.Error(err))
	}

	logger.Info("服务已安全退出")
}

// registerBuiltinTools 注册所有内置工具
func registerBuiltinTools(registry *tool.Registry, logger *zap.Logger) {
	tools := []tool.Tool{
		toolbuiltin.NewSearchTool(logger),
		toolbuiltin.NewCalculatorTool(logger),
		toolbuiltin.NewDatabaseTool(logger),
	}
	for _, t := range tools {
		if err := registry.Register(t); err != nil {
			logger.Warn("注册工具失败", zap.String("tool", t.Name()), zap.Error(err))
		}
	}
}

// initLogger 根据配置初始化 zap 日志
func initLogger(cfg config.LogConfig) *zap.Logger {
	var zapCfg zap.Config
	if cfg.Format == "json" {
		zapCfg = zap.NewProductionConfig()
	} else {
		zapCfg = zap.NewDevelopmentConfig()
	}

	level, err := zap.ParseAtomicLevel(cfg.Level)
	if err == nil {
		zapCfg.Level = level
	}

	logger, err := zapCfg.Build()
	if err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	return logger
}
