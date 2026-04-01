package com.agent.platform.service.agent;

import com.agent.platform.infrastructure.llm.ModelRouter;
import com.agent.platform.infrastructure.trace.TraceService;
import com.agent.platform.model.dto.ChatResponse;
import com.agent.platform.service.tool.BaseTool;
import com.agent.platform.service.tool.ToolRegistry;
import com.agent.platform.service.tool.ToolRouter;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.ai.chat.prompt.Prompt;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.util.*;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * ReAct Agent
 * <p>
 * 实现 Reasoning + Acting 循环：
 * <ol>
 *   <li>Thought：思考当前问题，分析需要什么信息</li>
 *   <li>Action：选择并调用合适的工具</li>
 *   <li>Observation：观察工具返回结果</li>
 *   <li>重复直到能够给出最终答案</li>
 * </ol>
 *
 * @see <a href="https://arxiv.org/abs/2210.03629">ReAct: Synergizing Reasoning and Acting</a>
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class ReActAgent {

    private final ModelRouter modelRouter;
    private final ToolRegistry toolRegistry;
    private final ToolRouter toolRouter;
    private final TraceService traceService;

    @Value("${agent.orchestrator.max-iterations:10}")
    private int maxIterations;

    private static final String REACT_SYSTEM_PROMPT = """
            你是一个智能助手，使用 ReAct（推理+行动）方式来回答用户的问题。
            
            %s
            
            请严格按照以下格式思考和行动：
            
            Thought: [分析当前情况，思考下一步应该做什么]
            Action: [要调用的工具名称]
            Action Input: [工具的参数，JSON格式]
            
            当你观察到工具返回的结果后，继续思考：
            
            Thought: [根据观察结果进行分析]
            Action: [如果需要，继续调用其他工具]
            Action Input: [工具参数]
            
            当你有足够的信息来回答用户问题时，使用：
            
            Thought: [最终分析]
            Final Answer: [给用户的最终回答]
            
            重要规则：
            1. 每次只调用一个工具
            2. 仔细分析每次工具返回的结果
            3. 如果工具调用失败，尝试其他方式
            4. 最终回答要完整、准确、有帮助
            """;

    private static final Pattern ACTION_PATTERN = Pattern.compile(
            "Action:\\s*(.+?)\\s*\nAction Input:\\s*(.+?)(?=\\n|$)", Pattern.DOTALL);
    private static final Pattern FINAL_ANSWER_PATTERN = Pattern.compile(
            "Final Answer:\\s*(.+)", Pattern.DOTALL);
    private static final Pattern THOUGHT_PATTERN = Pattern.compile(
            "Thought:\\s*(.+?)(?=\\nAction|\\nFinal|$)", Pattern.DOTALL);

    /**
     * 执行 ReAct 循环
     *
     * @param query         用户查询
     * @param context       附加上下文（如 RAG 检索结果）
     * @param availableTools 可用工具列表
     * @param traceId       追踪 ID
     * @return 包含思考过程的完整响应
     */
    public ReActResult execute(String query, String context, List<String> availableTools, String traceId) {
        log.info("ReAct Agent 开始执行: query={}", query);

        String toolDescriptions = toolRegistry.buildToolDescriptions(availableTools);
        String systemPrompt = String.format(REACT_SYSTEM_PROMPT, toolDescriptions);

        StringBuilder conversationBuffer = new StringBuilder();
        conversationBuffer.append("用户问题: ").append(query).append("\n\n");

        if (context != null && !context.isBlank()) {
            conversationBuffer.append("参考信息:\n").append(context).append("\n\n");
        }

        List<ChatResponse.ThinkingStep> thinkingSteps = new ArrayList<>();
        List<String> usedTools = new ArrayList<>();

        for (int i = 0; i < maxIterations; i++) {
            log.debug("ReAct 迭代 {}/{}", i + 1, maxIterations);

            traceService.addSpan(traceId, "react_iteration_" + (i + 1),
                    Map.of("iteration", i + 1));

            String fullPrompt = systemPrompt + "\n\n" + conversationBuffer;
            org.springframework.ai.chat.model.ChatResponse aiResponse =
                    modelRouter.call(new Prompt(fullPrompt), null);
            String llmOutput = aiResponse.getResult().getOutput().getText();

            log.debug("LLM 输出:\n{}", llmOutput);

            Matcher thoughtMatcher = THOUGHT_PATTERN.matcher(llmOutput);
            String thought = thoughtMatcher.find() ? thoughtMatcher.group(1).trim() : "";

            Matcher finalMatcher = FINAL_ANSWER_PATTERN.matcher(llmOutput);
            if (finalMatcher.find()) {
                String finalAnswer = finalMatcher.group(1).trim();

                thinkingSteps.add(ChatResponse.ThinkingStep.builder()
                        .step(i + 1)
                        .thought(thought)
                        .action("Final Answer")
                        .observation(finalAnswer)
                        .build());

                log.info("ReAct Agent 完成，共 {} 次迭代", i + 1);

                return ReActResult.builder()
                        .finalAnswer(finalAnswer)
                        .thinkingSteps(thinkingSteps)
                        .usedTools(usedTools)
                        .iterations(i + 1)
                        .build();
            }

            Matcher actionMatcher = ACTION_PATTERN.matcher(llmOutput);
            if (actionMatcher.find()) {
                String toolName = actionMatcher.group(1).trim();
                String toolInput = actionMatcher.group(2).trim();

                log.info("ReAct 调用工具: tool={}, input={}", toolName, toolInput);

                BaseTool.ToolResult toolResult = toolRouter.route(toolName, toolInput);
                usedTools.add(toolName);

                String observation = toolResult.success()
                        ? toolResult.output()
                        : "工具调用失败: " + toolResult.output();

                thinkingSteps.add(ChatResponse.ThinkingStep.builder()
                        .step(i + 1)
                        .thought(thought)
                        .action(toolName + "(" + toolInput + ")")
                        .observation(observation)
                        .build());

                conversationBuffer.append(llmOutput).append("\n");
                conversationBuffer.append("Observation: ").append(observation).append("\n\n");

                traceService.addSpan(traceId, "tool_call",
                        Map.of("tool", toolName, "success", toolResult.success(),
                                "elapsed_ms", toolResult.elapsedMs()));
            } else {
                log.warn("LLM 输出格式异常，无法解析 Action 或 Final Answer");
                return ReActResult.builder()
                        .finalAnswer(llmOutput)
                        .thinkingSteps(thinkingSteps)
                        .usedTools(usedTools)
                        .iterations(i + 1)
                        .build();
            }
        }

        log.warn("ReAct Agent 达到最大迭代次数 {}", maxIterations);
        return ReActResult.builder()
                .finalAnswer("抱歉，我在多次尝试后仍无法完全回答您的问题。以下是我目前的分析结果。")
                .thinkingSteps(thinkingSteps)
                .usedTools(usedTools)
                .iterations(maxIterations)
                .build();
    }

    @lombok.Data
    @lombok.Builder
    public static class ReActResult {
        private String finalAnswer;
        private List<ChatResponse.ThinkingStep> thinkingSteps;
        private List<String> usedTools;
        private int iterations;
    }
}
