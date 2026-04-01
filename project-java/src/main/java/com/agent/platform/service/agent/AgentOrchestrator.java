package com.agent.platform.service.agent;

import com.agent.platform.infrastructure.llm.ModelRouter;
import com.agent.platform.infrastructure.trace.TraceService;
import com.agent.platform.model.dto.ChatRequest;
import com.agent.platform.model.dto.ChatResponse;
import com.agent.platform.model.enums.AgentMode;
import com.agent.platform.service.intent.IntentRecognizer;
import com.agent.platform.service.memory.MemoryManager;
import com.agent.platform.service.rag.MultiRetriever;
import com.agent.platform.service.rag.RAGGenerator;
import com.agent.platform.service.rag.Reranker;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.ai.chat.prompt.Prompt;
import org.springframework.stereotype.Service;
import reactor.core.publisher.Flux;

import java.util.*;
import java.util.stream.Collectors;

/**
 * Agent 编排器
 * <p>
 * 系统的核心调度中枢，负责：
 * <ul>
 *   <li>意图识别 → 选择 Agent 模式</li>
 *   <li>RAG 检索 → 提供上下文</li>
 *   <li>Agent 执行 → ReAct / Planner / Reflection</li>
 *   <li>记忆管理 → 保存对话历史</li>
 *   <li>全链路追踪 → 记录执行过程</li>
 * </ul>
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class AgentOrchestrator {

    private final IntentRecognizer intentRecognizer;
    private final MemoryManager memoryManager;
    private final MultiRetriever multiRetriever;
    private final Reranker reranker;
    private final RAGGenerator ragGenerator;
    private final ReActAgent reActAgent;
    private final PlannerAgent plannerAgent;
    private final ReflectionAgent reflectionAgent;
    private final ModelRouter modelRouter;
    private final TraceService traceService;

    /**
     * 处理对话请求（同步）
     *
     * @param request 对话请求
     * @return 对话响应
     */
    public ChatResponse chat(ChatRequest request) {
        String traceId = traceService.startTrace("chat");

        try {
            String conversationId = resolveConversationId(request.getConversationId());

            memoryManager.saveUserMessage(conversationId, request.getMessage());

            traceService.addSpan(traceId, "intent_recognition", Map.of("message", request.getMessage()));
            IntentRecognizer.IntentResult intent = intentRecognizer.recognize(request.getMessage());

            AgentMode mode = request.getMode() != null ? request.getMode() : intent.getAgentMode();
            log.info("选定 Agent 模式: {} (意图: {}, 置信度: {})",
                    mode, intent.getIntentType(), intent.getConfidence());

            String ragContext = "";
            List<ChatResponse.SourceDocument> sources = new ArrayList<>();
            if (request.isEnableRag() && intent.isNeedsRag()) {
                traceService.addSpan(traceId, "rag_retrieval", Map.of());
                ragContext = performRAG(request.getMessage(), sources, traceId);
            }

            traceService.addSpan(traceId, "agent_execution",
                    Map.of("mode", mode.name(), "intent", intent.getIntentType()));

            ChatResponse response = dispatchToAgent(mode, request, ragContext, traceId);
            response.setConversationId(conversationId);
            response.setSources(sources.isEmpty() ? null : sources);
            response.setTraceId(traceId);

            memoryManager.saveAssistantMessage(conversationId, response.getReply());
            memoryManager.archiveIfNeeded(conversationId);

            return response;
        } catch (Exception e) {
            log.error("对话处理异常", e);
            return ChatResponse.builder()
                    .reply("抱歉，处理您的请求时发生了错误: " + e.getMessage())
                    .traceId(traceId)
                    .build();
        } finally {
            traceService.endTrace(traceId);
        }
    }

    /**
     * 处理对话请求（流式 SSE）
     */
    public Flux<String> chatStream(ChatRequest request) {
        String traceId = traceService.startTrace("chat_stream");
        String conversationId = resolveConversationId(request.getConversationId());

        memoryManager.saveUserMessage(conversationId, request.getMessage());

        String ragContext = "";
        if (request.isEnableRag()) {
            IntentRecognizer.IntentResult intent = intentRecognizer.recognize(request.getMessage());
            if (intent.isNeedsRag()) {
                ragContext = performRAG(request.getMessage(), new ArrayList<>(), traceId);
            }
        }

        String promptText = buildDirectPrompt(request.getMessage(), ragContext);
        String forceModel = request.getModelOptions() != null ? request.getModelOptions().getModel() : null;

        return modelRouter.stream(new Prompt(promptText), forceModel)
                .map(chatResponse -> {
                    if (chatResponse.getResult() != null &&
                            chatResponse.getResult().getOutput() != null) {
                        String text = chatResponse.getResult().getOutput().getText();
                        return text != null ? text : "";
                    }
                    return "";
                })
                .filter(text -> !text.isEmpty())
                .doOnComplete(() -> {
                    traceService.endTrace(traceId);
                    log.info("流式对话完成: traceId={}", traceId);
                })
                .doOnError(e -> {
                    log.error("流式对话异常: traceId={}", traceId, e);
                    traceService.endTrace(traceId);
                });
    }

    /**
     * 执行 RAG 检索，获取相关上下文
     */
    private String performRAG(String query, List<ChatResponse.SourceDocument> sources, String traceId) {
        try {
            List<MultiRetriever.RetrievalResult> results = multiRetriever.retrieve(query, null);

            if (!results.isEmpty()) {
                results = reranker.rerank(query, results);
            }

            for (MultiRetriever.RetrievalResult r : results) {
                sources.add(ChatResponse.SourceDocument.builder()
                        .documentId(r.getDocumentId())
                        .content(r.getContent())
                        .score(r.getScore())
                        .build());
            }

            traceService.addSpan(traceId, "rag_results",
                    Map.of("count", results.size()));

            return results.stream()
                    .map(MultiRetriever.RetrievalResult::getContent)
                    .collect(Collectors.joining("\n\n---\n\n"));
        } catch (Exception e) {
            log.warn("RAG 检索失败，降级为无上下文模式: {}", e.getMessage());
            return "";
        }
    }

    /**
     * 根据模式分发到对应的 Agent
     */
    private ChatResponse dispatchToAgent(AgentMode mode, ChatRequest request,
                                          String ragContext, String traceId) {
        return switch (mode) {
            case REACT -> {
                ReActAgent.ReActResult result = reActAgent.execute(
                        request.getMessage(), ragContext, request.getTools(), traceId);
                yield ChatResponse.builder()
                        .reply(result.getFinalAnswer())
                        .thinkingSteps(result.getThinkingSteps())
                        .usedTools(result.getUsedTools())
                        .build();
            }
            case PLANNER -> {
                PlannerAgent.PlanResult result = plannerAgent.execute(
                        request.getMessage(), ragContext, request.getTools(), traceId);
                yield ChatResponse.builder()
                        .reply(result.getFinalAnswer())
                        .thinkingSteps(result.getThinkingSteps())
                        .usedTools(result.getUsedTools())
                        .build();
            }
            case REFLECTION -> {
                ReflectionAgent.ReflectionResult result = reflectionAgent.execute(
                        request.getMessage(), ragContext, traceId);
                yield ChatResponse.builder()
                        .reply(result.getFinalAnswer())
                        .thinkingSteps(result.getThinkingSteps())
                        .build();
            }
            case DIRECT -> {
                String reply = directChat(request.getMessage(), ragContext);
                yield ChatResponse.builder()
                        .reply(reply)
                        .build();
            }
        };
    }

    /**
     * 直接对话模式（不经过 Agent 编排）
     */
    private String directChat(String message, String context) {
        String promptText = buildDirectPrompt(message, context);
        return modelRouter.call(new Prompt(promptText), null)
                .getResult().getOutput().getText();
    }

    private String buildDirectPrompt(String message, String context) {
        if (context != null && !context.isBlank()) {
            return String.format("""
                    你是一个专业的 AI 助手。请基于以下参考信息回答用户的问题。
                    
                    参考信息：
                    %s
                    
                    用户问题：%s
                    """, context, message);
        }
        return "你是一个专业的 AI 助手。请回答用户的问题。\n\n用户问题：" + message;
    }

    private String resolveConversationId(String conversationId) {
        if (conversationId == null || conversationId.isBlank()) {
            return UUID.randomUUID().toString().replace("-", "");
        }
        return conversationId;
    }
}
