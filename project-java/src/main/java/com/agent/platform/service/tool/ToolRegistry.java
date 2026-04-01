package com.agent.platform.service.tool;

import jakarta.annotation.PostConstruct;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.stream.Collectors;

/**
 * 工具注册中心
 * <p>
 * 管理所有可用工具的注册、查询、生命周期。
 * 启动时自动扫描所有 {@link BaseTool} 实现并注册，
 * 同时支持运行时动态注册/注销。
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class ToolRegistry {

    /** 工具注册表：toolName → BaseTool 实例 */
    private final Map<String, BaseTool> registry = new ConcurrentHashMap<>();

    /** Spring 容器中所有 BaseTool 实现（自动注入） */
    private final List<BaseTool> autoDiscoveredTools;

    @PostConstruct
    public void init() {
        for (BaseTool tool : autoDiscoveredTools) {
            register(tool);
        }
        log.info("工具注册中心初始化完成，共注册 {} 个工具: {}",
                registry.size(), registry.keySet());
    }

    /**
     * 注册工具
     */
    public void register(BaseTool tool) {
        String name = tool.getName();
        if (registry.containsKey(name)) {
            log.warn("工具名称冲突，覆盖已有注册: {}", name);
        }
        registry.put(name, tool);
        log.info("注册工具: name={}, description={}", name, tool.getDescription());
    }

    /**
     * 注销工具
     */
    public void unregister(String toolName) {
        BaseTool removed = registry.remove(toolName);
        if (removed != null) {
            log.info("注销工具: {}", toolName);
        }
    }

    /**
     * 根据名称获取工具
     */
    public Optional<BaseTool> getTool(String name) {
        return Optional.ofNullable(registry.get(name));
    }

    /**
     * 获取所有已注册工具
     */
    public Collection<BaseTool> getAllTools() {
        return Collections.unmodifiableCollection(registry.values());
    }

    /**
     * 获取指定子集的工具
     *
     * @param toolNames 工具名称列表，为 null 则返回全部
     */
    public List<BaseTool> getTools(List<String> toolNames) {
        if (toolNames == null || toolNames.isEmpty()) {
            return new ArrayList<>(registry.values());
        }
        return toolNames.stream()
                .map(registry::get)
                .filter(Objects::nonNull)
                .collect(Collectors.toList());
    }

    /**
     * 生成工具描述列表（供 LLM Prompt 使用）
     */
    public String buildToolDescriptions(List<String> toolNames) {
        List<BaseTool> tools = getTools(toolNames);

        StringBuilder sb = new StringBuilder();
        sb.append("可用工具列表:\n\n");
        for (BaseTool tool : tools) {
            sb.append("工具名: ").append(tool.getName()).append("\n");
            sb.append("描述: ").append(tool.getDescription()).append("\n");
            sb.append("参数: ").append(tool.getParameterSchema()).append("\n\n");
        }
        return sb.toString();
    }

    /**
     * 判断工具是否已注册
     */
    public boolean contains(String toolName) {
        return registry.containsKey(toolName);
    }
}
