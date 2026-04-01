package com.agent.platform.service.rag;

import com.agent.platform.infrastructure.cache.RedisCacheService;
import com.agent.platform.infrastructure.vectordb.MilvusService;
import io.milvus.grpc.SearchResults;
import io.milvus.response.SearchResultsWrapper;
import lombok.Builder;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.ai.openai.OpenAiChatModel;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.*;

/**
 * 多路检索引擎
 * <p>
 * 支持多种检索策略并行执行，合并结果后统一排序。
 * <ul>
 *   <li>向量语义检索：通过 Milvus 进行余弦相似度搜索</li>
 *   <li>关键词检索：基于 BM25 或简单关键词匹配</li>
 *   <li>缓存层：高频查询结果缓存，避免重复检索</li>
 * </ul>
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class MultiRetriever {

    private final MilvusService milvusService;
    private final RedisCacheService cacheService;

    @Qualifier("fallbackChatModel")
    private final OpenAiChatModel embeddingModel;

    @Value("${agent.rag.top-k:5}")
    private int topK;

    @Value("${agent.rag.similarity-threshold:0.7}")
    private double similarityThreshold;

    private static final String CACHE_PREFIX = "rag:retrieve:";
    private static final Duration CACHE_TTL = Duration.ofMinutes(30);

    /**
     * 执行多路检索
     *
     * @param query       用户查询
     * @param queryVector 查询向量（如果为 null 会尝试生成）
     * @return 检索结果列表
     */
    public List<RetrievalResult> retrieve(String query, List<Float> queryVector) {
        log.info("开始多路检索: query={}, topK={}", truncate(query, 80), topK);

        Optional<List<RetrievalResult>> cached = checkCache(query);
        if (cached.isPresent()) {
            log.info("命中检索缓存: query={}", truncate(query, 80));
            return cached.get();
        }

        List<RetrievalResult> allResults = new ArrayList<>();

        if (queryVector != null && !queryVector.isEmpty()) {
            List<RetrievalResult> vectorResults = vectorRetrieval(queryVector);
            allResults.addAll(vectorResults);
            log.info("向量检索返回 {} 条结果", vectorResults.size());
        }

        List<RetrievalResult> keywordResults = keywordRetrieval(query);
        allResults.addAll(keywordResults);
        log.info("关键词检索返回 {} 条结果", keywordResults.size());

        List<RetrievalResult> merged = mergeAndDeduplicate(allResults);

        List<RetrievalResult> filtered = merged.stream()
                .filter(r -> r.getScore() >= similarityThreshold)
                .sorted(Comparator.comparingDouble(RetrievalResult::getScore).reversed())
                .limit(topK)
                .toList();

        cacheResults(query, filtered);

        log.info("多路检索完成: 合并后 {} 条, 过滤后 {} 条", merged.size(), filtered.size());
        return filtered;
    }

    /**
     * 向量语义检索
     */
    private List<RetrievalResult> vectorRetrieval(List<Float> queryVector) {
        try {
            SearchResults searchResults = milvusService.searchSimilar(queryVector, topK);
            SearchResultsWrapper wrapper = new SearchResultsWrapper(searchResults.getResults());

            List<RetrievalResult> results = new ArrayList<>();
            for (int i = 0; i < wrapper.getRowRecords(0).size(); i++) {
                SearchResultsWrapper.IDScore idScore = wrapper.getIDScore(0).get(i);
                Map<String, Object> fieldValues = wrapper.getRowRecords(0).get(i).getFieldValues();

                results.add(RetrievalResult.builder()
                        .documentId(fieldValues.getOrDefault("doc_id", "").toString())
                        .content(fieldValues.getOrDefault("content", "").toString())
                        .score((double) idScore.getScore())
                        .source("vector")
                        .build());
            }
            return results;
        } catch (Exception e) {
            log.error("向量检索失败", e);
            return Collections.emptyList();
        }
    }

    /**
     * 关键词检索（简化实现，生产环境可对接 Elasticsearch）
     */
    private List<RetrievalResult> keywordRetrieval(String query) {
        // 简化实现：生产环境应对接 ES 或 MySQL 全文索引
        return Collections.emptyList();
    }

    /**
     * 合并去重：相同文档取最高分
     */
    private List<RetrievalResult> mergeAndDeduplicate(List<RetrievalResult> results) {
        Map<String, RetrievalResult> deduped = new LinkedHashMap<>();

        for (RetrievalResult result : results) {
            String key = result.getDocumentId();
            if (deduped.containsKey(key)) {
                RetrievalResult existing = deduped.get(key);
                if (result.getScore() > existing.getScore()) {
                    deduped.put(key, result);
                }
            } else {
                deduped.put(key, result);
            }
        }

        return new ArrayList<>(deduped.values());
    }

    private Optional<List<RetrievalResult>> checkCache(String query) {
        // 简化实现：通过查询文本 hash 做缓存 key
        return Optional.empty();
    }

    private void cacheResults(String query, List<RetrievalResult> results) {
        // 生产环境应实现缓存写入
    }

    private String truncate(String text, int maxLen) {
        return text.length() <= maxLen ? text : text.substring(0, maxLen) + "...";
    }

    @Data
    @Builder
    public static class RetrievalResult {
        private String documentId;
        private String content;
        private double score;
        /** 来源标识：vector / keyword / hybrid */
        private String source;
    }
}
