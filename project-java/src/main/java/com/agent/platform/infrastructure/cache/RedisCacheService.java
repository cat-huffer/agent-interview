package com.agent.platform.infrastructure.cache;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.List;
import java.util.Optional;

/**
 * Redis 缓存服务
 * <p>
 * 提供类型安全的缓存操作，支持对象序列化/反序列化
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class RedisCacheService {

    private final StringRedisTemplate redisTemplate;
    private final ObjectMapper objectMapper;

    private static final String KEY_PREFIX = "agent:";

    /**
     * 写入缓存（带过期时间）
     */
    public <T> void set(String key, T value, Duration ttl) {
        try {
            String json = objectMapper.writeValueAsString(value);
            redisTemplate.opsForValue().set(buildKey(key), json, ttl);
            log.debug("写入缓存: key={}, ttl={}s", key, ttl.getSeconds());
        } catch (JsonProcessingException e) {
            log.error("缓存序列化失败: key={}", key, e);
        }
    }

    /**
     * 读取缓存
     */
    public <T> Optional<T> get(String key, Class<T> clazz) {
        String json = redisTemplate.opsForValue().get(buildKey(key));
        if (json == null) {
            return Optional.empty();
        }

        try {
            T value = objectMapper.readValue(json, clazz);
            log.debug("命中缓存: key={}", key);
            return Optional.of(value);
        } catch (JsonProcessingException e) {
            log.error("缓存反序列化失败: key={}", key, e);
            return Optional.empty();
        }
    }

    /**
     * 删除缓存
     */
    public void delete(String key) {
        redisTemplate.delete(buildKey(key));
        log.debug("删除缓存: key={}", key);
    }

    /**
     * 右侧追加到列表（用于消息历史）
     */
    public void listRightPush(String key, String value) {
        redisTemplate.opsForList().rightPush(buildKey(key), value);
    }

    /**
     * 获取列表范围
     */
    public List<String> listRange(String key, long start, long end) {
        return redisTemplate.opsForList().range(buildKey(key), start, end);
    }

    /**
     * 裁剪列表，只保留指定范围
     */
    public void listTrim(String key, long start, long end) {
        redisTemplate.opsForList().trim(buildKey(key), start, end);
    }

    /**
     * 设置过期时间
     */
    public void expire(String key, Duration ttl) {
        redisTemplate.expire(buildKey(key), ttl);
    }

    /**
     * 判断 key 是否存在
     */
    public boolean exists(String key) {
        return Boolean.TRUE.equals(redisTemplate.hasKey(buildKey(key)));
    }

    private String buildKey(String key) {
        return KEY_PREFIX + key;
    }
}
