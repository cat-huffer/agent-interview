package com.agent.platform.controller;

import com.agent.platform.common.Result;
import com.agent.platform.model.dto.ChatRequest;
import com.agent.platform.model.dto.ChatResponse;
import com.agent.platform.service.agent.AgentOrchestrator;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Flux;

import java.time.Duration;

/**
 * 对话接口控制器
 * <p>
 * 提供普通对话和 SSE 流式对话两种模式
 */
@Slf4j
@RestController
@RequestMapping("/api/v1/chat")
@RequiredArgsConstructor
public class ChatController {

    private final AgentOrchestrator agentOrchestrator;

    /**
     * 普通对话接口
     * <p>
     * 同步返回完整的对话结果，包含思考过程和工具调用记录
     */
    @PostMapping
    public Result<ChatResponse> chat(@Valid @RequestBody ChatRequest request) {
        log.info("收到对话请求: conversationId={}, mode={}, message={}",
                request.getConversationId(),
                request.getMode(),
                truncate(request.getMessage(), 100));

        ChatResponse response = agentOrchestrator.chat(request);

        log.info("对话完成: conversationId={}, traceId={}",
                response.getConversationId(), response.getTraceId());

        return Result.success(response).withTraceId(response.getTraceId());
    }

    /**
     * SSE 流式对话接口
     * <p>
     * 通过 Server-Sent Events 实时推送生成内容，
     * 前端可通过 EventSource 或 fetch + ReadableStream 消费。
     * <p>
     * 响应格式：每个 SSE 事件的 data 字段包含一个文本片段，
     * 流结束时发送 "[DONE]" 标记。
     */
    @PostMapping(value = "/stream", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
    public Flux<String> chatStream(@Valid @RequestBody ChatRequest request) {
        log.info("收到流式对话请求: conversationId={}, message={}",
                request.getConversationId(),
                truncate(request.getMessage(), 100));

        return agentOrchestrator.chatStream(request)
                .timeout(Duration.ofSeconds(120))
                .concatWith(Flux.just("[DONE]"))
                .doOnError(e -> log.error("流式对话异常", e))
                .onErrorResume(e -> Flux.just("生成过程中发生错误: " + e.getMessage(), "[DONE]"));
    }

    private String truncate(String text, int maxLen) {
        if (text == null) return "";
        return text.length() <= maxLen ? text : text.substring(0, maxLen) + "...";
    }
}
