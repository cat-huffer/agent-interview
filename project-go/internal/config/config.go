// Package config 提供应用程序的配置管理能力。
// 支持从配置文件（YAML）和环境变量中加载配置，环境变量优先级更高。
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 应用程序全局配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	LLM      LLMConfig      `mapstructure:"llm"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Milvus   MilvusConfig   `mapstructure:"milvus"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	Agent    AgentConfig    `mapstructure:"agent"`
	RAG      RAGConfig      `mapstructure:"rag"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	Mode         string        `mapstructure:"mode"` // debug / release / test
}

// LLMConfig 大语言模型客户端配置
type LLMConfig struct {
	Models          []ModelConfig  `mapstructure:"models"`
	DefaultModel    string         `mapstructure:"default_model"`
	RequestTimeout  time.Duration  `mapstructure:"request_timeout"`
	CircuitBreaker  CBConfig       `mapstructure:"circuit_breaker"`
}

// ModelConfig 单个模型的配置
type ModelConfig struct {
	Name     string `mapstructure:"name"`
	Provider string `mapstructure:"provider"` // openai / anthropic / local
	APIKey   string `mapstructure:"api_key"`
	BaseURL  string `mapstructure:"base_url"`
	Model    string `mapstructure:"model"`
	Priority int    `mapstructure:"priority"` // 优先级，数值越小优先级越高
}

// CBConfig 熔断器配置
type CBConfig struct {
	FailureThreshold int           `mapstructure:"failure_threshold"` // 触发熔断的连续失败次数
	SuccessThreshold int           `mapstructure:"success_threshold"` // 半开状态下恢复所需的连续成功次数
	Timeout          time.Duration `mapstructure:"timeout"`           // 熔断器打开后的冷却时间
}

// RedisConfig Redis 连接配置
type RedisConfig struct {
	Addr         string        `mapstructure:"addr"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// MilvusConfig Milvus 向量数据库连接配置
type MilvusConfig struct {
	Addr           string `mapstructure:"addr"`
	CollectionName string `mapstructure:"collection_name"`
	Dimension      int    `mapstructure:"dimension"`
	MetricType     string `mapstructure:"metric_type"` // L2 / IP / COSINE
}

// PostgresConfig PostgreSQL 数据库连接配置
type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

// AgentConfig Agent 编排器配置
type AgentConfig struct {
	MaxIterations   int           `mapstructure:"max_iterations"`   // ReAct 最大迭代次数
	DefaultTimeout  time.Duration `mapstructure:"default_timeout"`  // 单次 Agent 执行超时
	EnableReflection bool         `mapstructure:"enable_reflection"` // 是否启用反思机制
}

// RAGConfig 检索增强生成配置
type RAGConfig struct {
	TopK           int     `mapstructure:"top_k"`            // 检索返回的文档数量
	ScoreThreshold float64 `mapstructure:"score_threshold"`  // 相似度阈值
	ChunkSize      int     `mapstructure:"chunk_size"`       // 文档分块大小
	ChunkOverlap   int     `mapstructure:"chunk_overlap"`    // 分块重叠长度
	EnableRerank   bool    `mapstructure:"enable_rerank"`    // 是否启用重排序
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug / info / warn / error
	Format string `mapstructure:"format"` // json / text
}

// DSN 返回 PostgreSQL 连接字符串
func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.DBName, p.SSLMode,
	)
}

// Load 从配置文件和环境变量加载配置。
// 优先级：环境变量 > 配置文件 > 默认值。
// 环境变量前缀为 APP_，使用下划线分隔层级，例如 APP_SERVER_PORT=8080。
func Load(path string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 配置文件
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/ai-agent/")
	}

	// 环境变量绑定：APP_SERVER_PORT -> server.port
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 尝试读取配置文件（不存在时使用默认值 + 环境变量）
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return cfg, nil
}

// setDefaults 设置所有配置项的默认值
func setDefaults(v *viper.Viper) {
	// 服务器默认配置
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "60s")
	v.SetDefault("server.mode", "debug")

	// LLM 默认配置
	v.SetDefault("llm.default_model", "gpt-4")
	v.SetDefault("llm.request_timeout", "60s")
	v.SetDefault("llm.circuit_breaker.failure_threshold", 5)
	v.SetDefault("llm.circuit_breaker.success_threshold", 3)
	v.SetDefault("llm.circuit_breaker.timeout", "30s")

	// Redis 默认配置
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 20)
	v.SetDefault("redis.read_timeout", "3s")
	v.SetDefault("redis.write_timeout", "3s")

	// Milvus 默认配置
	v.SetDefault("milvus.addr", "localhost:19530")
	v.SetDefault("milvus.collection_name", "documents")
	v.SetDefault("milvus.dimension", 1536)
	v.SetDefault("milvus.metric_type", "COSINE")

	// PostgreSQL 默认配置
	v.SetDefault("postgres.host", "localhost")
	v.SetDefault("postgres.port", 5432)
	v.SetDefault("postgres.user", "postgres")
	v.SetDefault("postgres.password", "postgres")
	v.SetDefault("postgres.dbname", "ai_agent")
	v.SetDefault("postgres.ssl_mode", "disable")

	// Agent 默认配置
	v.SetDefault("agent.max_iterations", 10)
	v.SetDefault("agent.default_timeout", "120s")
	v.SetDefault("agent.enable_reflection", true)

	// RAG 默认配置
	v.SetDefault("rag.top_k", 5)
	v.SetDefault("rag.score_threshold", 0.7)
	v.SetDefault("rag.chunk_size", 512)
	v.SetDefault("rag.chunk_overlap", 64)
	v.SetDefault("rag.enable_rerank", true)

	// 日志默认配置
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
}
