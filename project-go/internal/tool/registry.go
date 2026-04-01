package tool

import (
	"fmt"
	"sync"
)

// Registry 工具注册中心。
// 使用读写锁保证并发安全，支持运行时动态注册和注销工具。
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry 创建工具注册中心
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register 注册一个工具。如果同名工具已存在则返回错误。
func (r *Registry) Register(t Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := t.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("工具 %s 已注册", name)
	}

	r.tools[name] = t
	return nil
}

// MustRegister 注册一个工具，如果失败则 panic。
// 适用于程序初始化阶段注册内置工具。
func (r *Registry) MustRegister(t Tool) {
	if err := r.Register(t); err != nil {
		panic(err)
	}
}

// Unregister 注销指定名称的工具
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// Get 根据名称获取工具
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tools[name]
	return t, ok
}

// List 返回所有已注册工具的名称列表
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// ListTools 返回所有已注册的工具实例
func (r *Registry) ListTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// Count 返回已注册工具的数量
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// ToToolDefs 将所有已注册工具转换为 LLM Function Calling 格式，用于发送给大模型
func (r *Registry) ToToolDefs() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]map[string]interface{}, 0, len(r.tools))
	for _, t := range r.tools {
		def := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        t.Name(),
				"description": t.Description(),
				"parameters":  t.Parameters(),
			},
		}
		defs = append(defs, def)
	}
	return defs
}
