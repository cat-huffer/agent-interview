package com.agent.platform.service.memory;

import com.agent.platform.infrastructure.cache.RedisCacheService;
import com.agent.platform.infrastructure.llm.ModelRouter;
import com.agent.platform.infrastructure.vectordb.MilvusService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.ai.chat.prompt.Prompt;
import org.springframework.ai.openai.OpenAiChatModel;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.stereotype.Component;

import java.time.Duration;
import java.util.List;
import java.util.Map;

/**
 * 长期记忆
 * <p>
 * 将重要的对话摘要和用户偏好向量化后存入 Milvus，
 * 用于跨会话的知识检索。通过 LLM 提取摘要后存储。
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class LongTermMemory {

    private final MilvusService milvusService;
    private final RedisCacheService cacheService;

    @Qualifier("fallbackChatModel")
    private final OpenAiChatModel chatModel;

    private static final String SUMMARY_KEY_PREFIX = "memory:long:summary:";

    /**
     * 将对话摘要存入长期记忆
     *
     * @param conversationId 会话 ID
     * @param summary        摘要文本
     * @param embedding      摘要的向量表示
     */
    public void store(String conversationId, String summary, List<Float> embedding) {
        milvusService.insertVectors(
                List.of(conversationId),
                List.of(embedding),
                List.of(summary)
        );

        cacheService.set(
                SUMMARY_KEY_PREFIX + conversationId,
                Map.of("summary", summary, "conversationId", conversationId),
                Duration.ofDays(30)
        );

        log.info("长期记忆存储成功: conversationId={}", conversationId);
    }

    /**
     * 根据查询内容检索相关的长期记忆
     *
     * @param queryEmbedding 查询向量
     * @param topK           返回数量
     * @return 搜索结果
     */
    public io.milvus.grpc.SearchResults recall(List<Float> queryEmbedding, int topK) {
        log.debug("检索长期记忆: topK={}", topK);
        return milvusService.searchSimilar(queryEmbedding, topK);
    }

    /**
     * 生成对话摘要（通过 LLM 提取关键信息）
     *
     * @param conversationHistory 对话历史
     * @return 摘要文本
     */
    public String generateSummary(List<String> conversationHistory) {
        String historyText = String.join("\n", conversationHistory);

        String promptText = String.format("""
                请将以下对话历史总结为简短的摘要，提取关键信息和用户偏好：
                
                对话历史：
                %s
                
                请输出摘要（不超过200字）：
                """, historyText);

        try {
            String summary = chatModel.call(new Prompt(promptText))
                    .getResult()
                    .getOutput()
                    .getText();
            log.info("生成对话摘要成功，长度: {}", summary.length());
            return summary;
        } catch (Exception e) {
            log.error("生成对话摘要失败", e);
            return "对话摘要生成失败";
        }
    }
}
