package com.agent.platform.service.tool;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

import java.util.Map;

/**
 * 工具路由器
 * <p>
 * 负责解析 LLM 返回的工具调用指令，路由到对应的工具执行。
 * 将 LLM 输出的 JSON 格式工具调用转化为实际的工具执行。
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class ToolRouter {

    private final ToolRegistry toolRegistry;
    private final ObjectMapper objectMapper;

    /**
     * 路由并执行工具调用
     *
     * @param toolName   工具名称
     * @param argsJson   参数 JSON 字符串
     * @return 执行结果
     */
    public BaseTool.ToolResult route(String toolName, String argsJson) {
        log.info("路由工具调用: tool={}, args={}", toolName, argsJson);

        return toolRegistry.getTool(toolName)
                .map(tool -> {
                    Map<String, Object> parameters = parseParameters(argsJson);
                    return tool.execute(parameters);
                })
                .orElseGet(() -> {
                    log.warn("未找到工具: {}", toolName);
                    return BaseTool.ToolResult.failure(
                            toolName,
                            "工具 '" + toolName + "' 未注册，可用工具: " + toolRegistry.getAllTools(),
                            0
                    );
                });
    }

    /**
     * 批量执行多个工具调用
     */
    public Map<String, BaseTool.ToolResult> routeMultiple(Map<String, String> toolCalls) {
        return toolCalls.entrySet().stream()
                .collect(java.util.stream.Collectors.toMap(
                        Map.Entry::getKey,
                        entry -> route(entry.getKey(), entry.getValue())
                ));
    }

    private Map<String, Object> parseParameters(String argsJson) {
        if (argsJson == null || argsJson.isBlank()) {
            return Map.of();
        }
        try {
            return objectMapper.readValue(argsJson, new TypeReference<>() {});
        } catch (JsonProcessingException e) {
            log.error("工具参数 JSON 解析失败: {}", argsJson, e);
            throw new IllegalArgumentException("工具参数格式错误: " + e.getMessage());
        }
    }
}
