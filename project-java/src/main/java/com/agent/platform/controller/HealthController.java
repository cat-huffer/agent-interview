package com.agent.platform.controller;

import com.agent.platform.common.Result;
import com.agent.platform.infrastructure.llm.CircuitBreaker;
import lombok.RequiredArgsConstructor;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.time.LocalDateTime;
import java.util.LinkedHashMap;
import java.util.Map;

/**
 * 健康检查控制器
 */
@RestController
@RequestMapping("/api/v1/health")
@RequiredArgsConstructor
public class HealthController {

    private final StringRedisTemplate redisTemplate;
    private final CircuitBreaker circuitBreaker;

    @GetMapping
    public Result<Map<String, Object>> health() {
        Map<String, Object> status = new LinkedHashMap<>();
        status.put("status", "UP");
        status.put("timestamp", LocalDateTime.now().toString());
        status.put("circuitBreaker", circuitBreaker.getState().name());
        status.put("redis", checkRedis());
        return Result.success(status);
    }

    private String checkRedis() {
        try {
            redisTemplate.opsForValue().get("health:check");
            return "UP";
        } catch (Exception e) {
            return "DOWN: " + e.getMessage();
        }
    }
}
