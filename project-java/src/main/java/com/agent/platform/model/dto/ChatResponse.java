package com.agent.platform.model.dto;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.List;

/**
 * 对话响应 DTO
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ChatResponse {

    /** 会话 ID */
    private String conversationId;

    /** 回复消息 */
    private String reply;

    /** Agent 执行的思考过程 */
    private List<ThinkingStep> thinkingSteps;

    /** 使用的工具列表 */
    private List<String> usedTools;

    /** RAG 检索的来源文档 */
    private List<SourceDocument> sources;

    /** 使用的模型信息 */
    private String model;

    /** Token 用量 */
    private TokenUsage tokenUsage;

    /** 追踪 ID */
    private String traceId;

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class ThinkingStep {
        private int step;
        private String thought;
        private String action;
        private String observation;
    }

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class SourceDocument {
        private String documentId;
        private String content;
        private double score;
    }

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class TokenUsage {
        private int promptTokens;
        private int completionTokens;
        private int totalTokens;
    }
}
