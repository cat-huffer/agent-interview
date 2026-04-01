package com.agent.platform.service.tool.builtin;

import com.agent.platform.service.tool.BaseTool;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

import java.util.Map;

/**
 * 搜索工具
 * <p>
 * 模拟外部搜索引擎调用，实际生产中应对接搜索 API
 */
@Slf4j
@Component
public class SearchTool extends BaseTool {

    @Override
    public String getName() {
        return "search";
    }

    @Override
    public String getDescription() {
        return "搜索互联网信息，获取最新的知识和事实。当用户问题涉及实时信息、最新事件、不确定的事实时使用。";
    }

    @Override
    public String getParameterSchema() {
        return """
                {
                  "type": "object",
                  "properties": {
                    "query": {
                      "type": "string",
                      "description": "搜索关键词"
                    }
                  },
                  "required": ["query"]
                }
                """;
    }

    @Override
    protected void validate(Map<String, Object> parameters) {
        if (!parameters.containsKey("query") || parameters.get("query").toString().isBlank()) {
            throw new IllegalArgumentException("搜索关键词不能为空");
        }
    }

    @Override
    protected String doExecute(Map<String, Object> parameters) {
        String query = parameters.get("query").toString();
        log.info("执行搜索: query={}", query);

        // 生产环境应对接真实搜索 API（如 SerpAPI、Bing API 等）
        return String.format("搜索 '%s' 的结果摘要：" +
                "根据搜索结果，找到了与 '%s' 相关的信息。" +
                "（此为模拟结果，生产环境需要对接真实搜索引擎）", query, query);
    }
}
