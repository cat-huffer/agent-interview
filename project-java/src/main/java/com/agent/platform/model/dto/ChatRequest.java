package com.agent.platform.model.dto;

import com.agent.platform.model.enums.AgentMode;
import jakarta.validation.constraints.NotBlank;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.List;

/**
 * 对话请求 DTO
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ChatRequest {

    /** 会话 ID，为空则新建会话 */
    private String conversationId;

    /** 用户消息 */
    @NotBlank(message = "消息内容不能为空")
    private String message;

    /** Agent 模式，默认 ReAct */
    @Builder.Default
    private AgentMode mode = AgentMode.REACT;

    /** 是否启用 RAG 检索 */
    @Builder.Default
    private boolean enableRag = true;

    /** 指定使用的工具列表，为空则使用全部可用工具 */
    private List<String> tools;

    /** 模型参数覆盖 */
    private ModelOptions modelOptions;

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class ModelOptions {
        private String model;
        private Double temperature;
        private Integer maxTokens;
    }
}
