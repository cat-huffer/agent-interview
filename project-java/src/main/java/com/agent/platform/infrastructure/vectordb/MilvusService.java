package com.agent.platform.infrastructure.vectordb;

import com.agent.platform.config.MilvusConfig;
import io.milvus.client.MilvusServiceClient;
import io.milvus.grpc.MutationResult;
import io.milvus.grpc.SearchResults;
import io.milvus.param.R;
import io.milvus.param.collection.HasCollectionParam;
import io.milvus.param.collection.LoadCollectionParam;
import io.milvus.param.dml.InsertParam;
import io.milvus.param.dml.SearchParam;
import jakarta.annotation.PostConstruct;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

/**
 * Milvus 向量数据库服务
 * <p>
 * 封装向量的插入、搜索、删除等操作
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class MilvusService {

    private final MilvusServiceClient milvusClient;
    private final MilvusConfig milvusConfig;

    @PostConstruct
    public void init() {
        ensureCollectionLoaded();
    }

    /**
     * 插入向量数据
     *
     * @param ids        文档 ID 列表
     * @param vectors    向量数据列表
     * @param contents   原始文本列表
     * @return 插入数量
     */
    public long insertVectors(List<String> ids, List<List<Float>> vectors, List<String> contents) {
        List<InsertParam.Field> fields = new ArrayList<>();
        fields.add(new InsertParam.Field("doc_id", ids));
        fields.add(new InsertParam.Field("embedding", vectors));
        fields.add(new InsertParam.Field("content", contents));

        InsertParam insertParam = InsertParam.newBuilder()
                .withCollectionName(milvusConfig.getCollectionName())
                .withFields(fields)
                .build();

        R<MutationResult> response = milvusClient.insert(insertParam);

        if (response.getStatus() != R.Status.Success.getCode()) {
            log.error("Milvus 向量插入失败: {}", response.getMessage());
            throw new RuntimeException("向量插入失败: " + response.getMessage());
        }

        long count = response.getData().getInsertCnt();
        log.info("成功插入 {} 条向量到集合 {}", count, milvusConfig.getCollectionName());
        return count;
    }

    /**
     * 向量相似度搜索
     *
     * @param queryVector 查询向量
     * @param topK        返回结果数量
     * @return 搜索结果
     */
    public SearchResults searchSimilar(List<Float> queryVector, int topK) {
        SearchParam searchParam = SearchParam.newBuilder()
                .withCollectionName(milvusConfig.getCollectionName())
                .withMetricType(io.milvus.param.MetricType.COSINE)
                .withOutFields(List.of("doc_id", "content"))
                .withTopK(topK)
                .withVectors(Collections.singletonList(queryVector))
                .withVectorFieldName("embedding")
                .withParams("{\"nprobe\": 16}")
                .build();

        R<SearchResults> response = milvusClient.search(searchParam);

        if (response.getStatus() != R.Status.Success.getCode()) {
            log.error("Milvus 向量搜索失败: {}", response.getMessage());
            throw new RuntimeException("向量搜索失败: " + response.getMessage());
        }

        log.debug("向量搜索完成，返回 {} 条结果", topK);
        return response.getData();
    }

    /**
     * 确保集合已加载到内存
     */
    private void ensureCollectionLoaded() {
        try {
            String collectionName = milvusConfig.getCollectionName();

            R<Boolean> hasCollection = milvusClient.hasCollection(
                    HasCollectionParam.newBuilder()
                            .withCollectionName(collectionName)
                            .build()
            );

            if (hasCollection.getData()) {
                milvusClient.loadCollection(
                        LoadCollectionParam.newBuilder()
                                .withCollectionName(collectionName)
                                .build()
                );
                log.info("Milvus 集合 {} 已加载到内存", collectionName);
            } else {
                log.warn("Milvus 集合 {} 不存在，需要先创建", collectionName);
            }
        } catch (Exception e) {
            log.warn("Milvus 初始化检查失败（服务可能未启动）: {}", e.getMessage());
        }
    }
}
