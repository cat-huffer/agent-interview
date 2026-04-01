package com.agent.platform.infrastructure.llm;

import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.time.Instant;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicReference;

/**
 * 三态熔断器
 * <p>
 * 状态转换：CLOSED → OPEN → HALF_OPEN → CLOSED
 * <ul>
 *   <li>CLOSED（关闭）：正常放行请求，失败计数达到阈值后转为 OPEN</li>
 *   <li>OPEN（打开）：拒绝所有请求，超时后转为 HALF_OPEN</li>
 *   <li>HALF_OPEN（半开）：放行部分探测请求，连续成功达阈值则关闭，任一失败则重新打开</li>
 * </ul>
 */
@Slf4j
@Component
public class CircuitBreaker {

    public enum State {
        CLOSED,
        OPEN,
        HALF_OPEN
    }

    private final AtomicReference<State> state = new AtomicReference<>(State.CLOSED);
    private final AtomicInteger failureCount = new AtomicInteger(0);
    private final AtomicInteger successCount = new AtomicInteger(0);
    private volatile Instant openedAt;

    @Value("${agent.circuit-breaker.failure-threshold:5}")
    private int failureThreshold;

    @Value("${agent.circuit-breaker.success-threshold:3}")
    private int successThreshold;

    @Value("${agent.circuit-breaker.timeout-seconds:30}")
    private int timeoutSeconds;

    /**
     * 判断当前是否允许请求通过
     */
    public boolean allowRequest() {
        State current = state.get();

        switch (current) {
            case CLOSED:
                return true;

            case OPEN:
                if (isTimeoutExpired()) {
                    if (state.compareAndSet(State.OPEN, State.HALF_OPEN)) {
                        log.info("熔断器状态转换: OPEN → HALF_OPEN，开始探测");
                        successCount.set(0);
                    }
                    return true;
                }
                log.debug("熔断器处于 OPEN 状态，拒绝请求");
                return false;

            case HALF_OPEN:
                return true;

            default:
                return true;
        }
    }

    /**
     * 记录请求成功
     */
    public void recordSuccess() {
        State current = state.get();

        if (current == State.HALF_OPEN) {
            int count = successCount.incrementAndGet();
            log.debug("HALF_OPEN 状态下成功计数: {}/{}", count, successThreshold);

            if (count >= successThreshold) {
                if (state.compareAndSet(State.HALF_OPEN, State.CLOSED)) {
                    log.info("熔断器状态转换: HALF_OPEN → CLOSED，服务恢复正常");
                    resetCounters();
                }
            }
        } else if (current == State.CLOSED) {
            failureCount.set(0);
        }
    }

    /**
     * 记录请求失败
     */
    public void recordFailure() {
        State current = state.get();

        if (current == State.HALF_OPEN) {
            if (state.compareAndSet(State.HALF_OPEN, State.OPEN)) {
                openedAt = Instant.now();
                log.warn("熔断器状态转换: HALF_OPEN → OPEN，探测失败，继续熔断");
            }
        } else if (current == State.CLOSED) {
            int count = failureCount.incrementAndGet();
            log.debug("CLOSED 状态下失败计数: {}/{}", count, failureThreshold);

            if (count >= failureThreshold) {
                if (state.compareAndSet(State.CLOSED, State.OPEN)) {
                    openedAt = Instant.now();
                    log.warn("熔断器状态转换: CLOSED → OPEN，连续失败 {} 次，触发熔断", count);
                }
            }
        }
    }

    /**
     * 获取当前熔断器状态
     */
    public State getState() {
        return state.get();
    }

    /**
     * 手动重置熔断器
     */
    public void reset() {
        state.set(State.CLOSED);
        resetCounters();
        log.info("熔断器已手动重置为 CLOSED");
    }

    private boolean isTimeoutExpired() {
        return openedAt != null &&
                Instant.now().isAfter(openedAt.plusSeconds(timeoutSeconds));
    }

    private void resetCounters() {
        failureCount.set(0);
        successCount.set(0);
    }
}
