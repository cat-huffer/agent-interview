package com.agent.platform.service.rag;

import com.agent.platform.infrastructure.llm.ModelRouter;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.ai.chat.prompt.Prompt;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;

/**
 * 重排序服务
 * <p>
 * 对初始检索结果进行二次排序，通过 LLM 评估每个文档
 * 与查询的相关性，提升最终结果的准确度。
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class Reranker {

    private final ModelRouter modelRouter;

    @Value("${agent.rag.rerank-top-n:3}")
    private int rerankTopN;

    private static final String RERANK_PROMPT = """
            请评估以下文档与用户查询的相关性，给出 0-10 的评分。
            
            用户查询: %s
            
            文档内容: %s
            
            请仅返回一个数字（0-10），表示相关性评分。10 表示完全相关，0 表示完全不相关。
            """;

    /**
     * 对检索结果进行重排序
     *
     * @param query   用户查询
     * @param results 初始检索结果
     * @return 重排后的结果
     */
    public List<MultiRetriever.RetrievalResult> rerank(String query, List<MultiRetriever.RetrievalResult> results) {
        if (results.size() <= rerankTopN) {
            log.debug("检索结果数量({})不超过 rerankTopN({}), 跳过重排", results.size(), rerankTopN);
            return results;
        }

        log.info("开始重排序: query={}, candidates={}", query, results.size());

        List<ScoredResult> scoredResults = new ArrayList<>();

        for (MultiRetriever.RetrievalResult result : results) {
            double rerankScore = scoreDocument(query, result.getContent());
            scoredResults.add(new ScoredResult(result, rerankScore));
        }

        List<MultiRetriever.RetrievalResult> reranked = scoredResults.stream()
                .sorted(Comparator.comparingDouble(ScoredResult::rerankScore).reversed())
                .limit(rerankTopN)
                .map(ScoredResult::result)
                .toList();

        log.info("重排序完成: 输入 {} 条, 输出 {} 条", results.size(), reranked.size());
        return reranked;
    }

    /**
     * 使用 LLM 为单个文档评分
     */
    private double scoreDocument(String query, String content) {
        try {
            String promptText = String.format(RERANK_PROMPT, query, truncate(content, 500));
            String scoreStr = modelRouter.call(new Prompt(promptText), "gpt-4o-mini")
                    .getResult()
                    .getOutput()
                    .getText()
                    .trim();

            return Double.parseDouble(scoreStr.replaceAll("[^0-9.]", ""));
        } catch (Exception e) {
            log.warn("LLM 重排评分失败: {}", e.getMessage());
            return 5.0;
        }
    }

    private String truncate(String text, int maxLen) {
        return text.length() <= maxLen ? text : text.substring(0, maxLen) + "...";
    }

    private record ScoredResult(MultiRetriever.RetrievalResult result, double rerankScore) {}
}
