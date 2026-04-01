package com.agent.platform.infrastructure.trace;

import lombok.Builder;
import lombok.Data;
import lombok.extern.slf4j.Slf4j;
import org.slf4j.MDC;
import org.springframework.stereotype.Service;

import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;

/**
 * 链路追踪服务
 * <p>
 * 为每个请求生成唯一 traceId，记录 Agent 执行全链路日志，
 * 便于调试、性能分析和问题排查
 */
@Slf4j
@Service
public class TraceService {

    private static final String TRACE_ID_KEY = "traceId";

    /** 存储活跃的追踪上下文（生产环境应接入分布式追踪系统） */
    private final Map<String, TraceContext> activeTraces = new ConcurrentHashMap<>();

    /**
     * 开始一个新的追踪
     */
    public String startTrace(String operation) {
        String traceId = UUID.randomUUID().toString().replace("-", "").substring(0, 16);
        MDC.put(TRACE_ID_KEY, traceId);

        TraceContext context = TraceContext.builder()
                .traceId(traceId)
                .operation(operation)
                .startTime(Instant.now())
                .spans(new ArrayList<>())
                .build();
        activeTraces.put(traceId, context);

        log.info("开始追踪: operation={}, traceId={}", operation, traceId);
        return traceId;
    }

    /**
     * 添加一个追踪节点（Span）
     */
    public void addSpan(String traceId, String spanName, Map<String, Object> attributes) {
        TraceContext context = activeTraces.get(traceId);
        if (context == null) {
            log.warn("追踪上下文不存在: traceId={}", traceId);
            return;
        }

        TraceSpan span = TraceSpan.builder()
                .name(spanName)
                .timestamp(Instant.now())
                .attributes(attributes)
                .build();
        context.getSpans().add(span);

        log.debug("[Trace:{}] Span: {} | {}", traceId, spanName, attributes);
    }

    /**
     * 结束追踪
     */
    public TraceContext endTrace(String traceId) {
        TraceContext context = activeTraces.remove(traceId);
        if (context != null) {
            context.setEndTime(Instant.now());
            context.setDurationMs(
                    context.getEndTime().toEpochMilli() - context.getStartTime().toEpochMilli()
            );
            log.info("追踪结束: traceId={}, duration={}ms, spans={}",
                    traceId, context.getDurationMs(), context.getSpans().size());
        }
        MDC.remove(TRACE_ID_KEY);
        return context;
    }

    /**
     * 获取当前追踪 ID
     */
    public String currentTraceId() {
        return MDC.get(TRACE_ID_KEY);
    }

    @Data
    @Builder
    public static class TraceContext {
        private String traceId;
        private String operation;
        private Instant startTime;
        private Instant endTime;
        private long durationMs;
        private List<TraceSpan> spans;
    }

    @Data
    @Builder
    public static class TraceSpan {
        private String name;
        private Instant timestamp;
        private Map<String, Object> attributes;
    }
}
