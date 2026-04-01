package com.agent.platform.service.memory;

import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.ai.chat.messages.AssistantMessage;
import org.springframework.ai.chat.messages.Message;
import org.springframework.ai.chat.messages.SystemMessage;
import org.springframework.ai.chat.messages.UserMessage;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.util.ArrayList;
import java.util.List;

/**
 * 记忆管理器
 * <p>
 * 统一管理短期记忆和长期记忆，负责：
 * <ul>
 *   <li>写入消息到短期记忆</li>
 *   <li>构建 LLM 所需的消息上下文</li>
 *   <li>触发长期记忆的归档</li>
 *   <li>记忆的生命周期管理</li>
 * </ul>
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class MemoryManager {

    private final ShortTermMemory shortTermMemory;
    private final LongTermMemory longTermMemory;

    @Value("${agent.memory.long-term.enabled:true}")
    private boolean longTermEnabled;

    @Value("${agent.memory.short-term.max-turns:20}")
    private int maxTurns;

    /**
     * 保存用户消息
     */
    public void saveUserMessage(String conversationId, String content) {
        shortTermMemory.addMessage(conversationId, "user", content);
        log.debug("保存用户消息到短期记忆: conversationId={}", conversationId);
    }

    /**
     * 保存助手回复
     */
    public void saveAssistantMessage(String conversationId, String content) {
        shortTermMemory.addMessage(conversationId, "assistant", content);
        log.debug("保存助手回复到短期记忆: conversationId={}", conversationId);
    }

    /**
     * 构建 LLM 请求所需的消息上下文
     * <p>
     * 组装顺序：系统提示 → 历史对话 → 长期记忆相关片段
     *
     * @param conversationId 会话 ID
     * @param systemPrompt   系统提示
     * @return 消息列表
     */
    public List<Message> buildContext(String conversationId, String systemPrompt) {
        List<Message> messages = new ArrayList<>();

        messages.add(new SystemMessage(systemPrompt));

        List<String> history = shortTermMemory.getRecentHistory(conversationId, maxTurns);
        for (String entry : history) {
            int separatorIdx = entry.indexOf(':');
            if (separatorIdx > 0) {
                String role = entry.substring(0, separatorIdx);
                String content = entry.substring(separatorIdx + 1);

                switch (role) {
                    case "user" -> messages.add(new UserMessage(content));
                    case "assistant" -> messages.add(new AssistantMessage(content));
                    default -> log.warn("未知的消息角色: {}", role);
                }
            }
        }

        log.debug("构建消息上下文: conversationId={}, totalMessages={}", conversationId, messages.size());
        return messages;
    }

    /**
     * 触发长期记忆归档
     * <p>
     * 当对话轮次超过阈值时，通过 LLM 生成摘要并存入向量数据库
     */
    public void archiveIfNeeded(String conversationId) {
        if (!longTermEnabled) {
            return;
        }

        List<String> history = shortTermMemory.getHistory(conversationId);
        if (history.size() < maxTurns * 2) {
            return;
        }

        try {
            String summary = longTermMemory.generateSummary(history);
            log.info("触发长期记忆归档: conversationId={}, summary={}", conversationId, summary);
            // 向量化需要 Embedding 模型，此处留作扩展
        } catch (Exception e) {
            log.error("长期记忆归档失败: conversationId={}", conversationId, e);
        }
    }

    /**
     * 清除会话所有记忆
     */
    public void clearMemory(String conversationId) {
        shortTermMemory.clear(conversationId);
        log.info("清除所有记忆: conversationId={}", conversationId);
    }

    /**
     * 获取原始对话历史
     */
    public List<String> getRawHistory(String conversationId) {
        return shortTermMemory.getHistory(conversationId);
    }
}
