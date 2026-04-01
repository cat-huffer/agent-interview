package com.agent.platform.service.intent;

import com.agent.platform.infrastructure.llm.ModelRouter;
import com.agent.platform.model.enums.AgentMode;
import lombok.Builder;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.ai.chat.prompt.Prompt;
import org.springframework.stereotype.Service;

import java.util.List;

/**
 * 意图识别服务
 * <p>
 * 分析用户输入的意图，决定使用哪种 Agent 模式和工具集合。
 * 支持基于规则的快速判断和基于 LLM 的深度理解。
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class IntentRecognizer {

    private final ModelRouter modelRouter;

    private static final String INTENT_PROMPT_TEMPLATE = """
            你是一个意图识别引擎。请分析用户的输入，判断用户的意图类别。
            
            意图类别：
            1. SIMPLE_CHAT - 简单闲聊、问候、闲谈
            2. KNOWLEDGE_QA - 知识问答，需要检索知识库
            3. TOOL_USE - 需要使用工具（搜索、计算、数据库查询等）
            4. COMPLEX_TASK - 复杂任务，需要多步推理和规划
            5. CODE_RELATED - 代码相关问题
            
            用户输入: %s
            
            请仅返回意图类别名称（如 SIMPLE_CHAT），不要返回其他内容。
            """;

    /**
     * 识别用户意图
     *
     * @param userInput 用户输入
     * @return 意图识别结果
     */
    public IntentResult recognize(String userInput) {
        log.info("开始意图识别: input={}", truncate(userInput, 100));

        IntentResult ruleResult = ruleBasedRecognize(userInput);
        if (ruleResult != null) {
            log.info("规则命中意图: type={}, confidence={}", ruleResult.getIntentType(), ruleResult.getConfidence());
            return ruleResult;
        }

        return llmBasedRecognize(userInput);
    }

    /**
     * 基于规则的快速意图判断
     */
    private IntentResult ruleBasedRecognize(String input) {
        String lower = input.toLowerCase().trim();

        if (lower.matches("^(你好|hi|hello|嗨|早上好|晚上好|hey).*")) {
            return IntentResult.builder()
                    .intentType("SIMPLE_CHAT")
                    .agentMode(AgentMode.DIRECT)
                    .confidence(0.95)
                    .needsRag(false)
                    .suggestedTools(List.of())
                    .build();
        }

        if (lower.contains("计算") || lower.matches(".*\\d+[+\\-*/]\\d+.*")) {
            return IntentResult.builder()
                    .intentType("TOOL_USE")
                    .agentMode(AgentMode.REACT)
                    .confidence(0.9)
                    .needsRag(false)
                    .suggestedTools(List.of("calculator"))
                    .build();
        }

        if (lower.contains("搜索") || lower.contains("查找") || lower.contains("最新")) {
            return IntentResult.builder()
                    .intentType("TOOL_USE")
                    .agentMode(AgentMode.REACT)
                    .confidence(0.85)
                    .needsRag(false)
                    .suggestedTools(List.of("search"))
                    .build();
        }

        if (lower.contains("查询数据") || lower.contains("sql") || lower.contains("数据库")) {
            return IntentResult.builder()
                    .intentType("TOOL_USE")
                    .agentMode(AgentMode.REACT)
                    .confidence(0.85)
                    .needsRag(false)
                    .suggestedTools(List.of("database_query"))
                    .build();
        }

        return null;
    }

    /**
     * 基于 LLM 的意图识别（兜底方案）
     */
    private IntentResult llmBasedRecognize(String input) {
        try {
            String promptText = String.format(INTENT_PROMPT_TEMPLATE, input);
            String intentType = modelRouter.call(new Prompt(promptText), "gpt-4o-mini")
                    .getResult()
                    .getOutput()
                    .getText()
                    .trim()
                    .toUpperCase();

            AgentMode mode = mapToAgentMode(intentType);
            boolean needsRag = "KNOWLEDGE_QA".equals(intentType);

            log.info("LLM 意图识别结果: type={}, mode={}", intentType, mode);

            return IntentResult.builder()
                    .intentType(intentType)
                    .agentMode(mode)
                    .confidence(0.8)
                    .needsRag(needsRag)
                    .suggestedTools(List.of())
                    .build();
        } catch (Exception e) {
            log.error("LLM 意图识别失败，使用默认意图", e);
            return IntentResult.builder()
                    .intentType("KNOWLEDGE_QA")
                    .agentMode(AgentMode.REACT)
                    .confidence(0.5)
                    .needsRag(true)
                    .suggestedTools(List.of())
                    .build();
        }
    }

    private AgentMode mapToAgentMode(String intentType) {
        return switch (intentType) {
            case "SIMPLE_CHAT" -> AgentMode.DIRECT;
            case "COMPLEX_TASK" -> AgentMode.PLANNER;
            default -> AgentMode.REACT;
        };
    }

    private String truncate(String text, int maxLength) {
        if (text.length() <= maxLength) return text;
        return text.substring(0, maxLength) + "...";
    }

    @Data
    @Builder
    public static class IntentResult {
        private String intentType;
        private AgentMode agentMode;
        private double confidence;
        private boolean needsRag;
        private List<String> suggestedTools;
    }
}
