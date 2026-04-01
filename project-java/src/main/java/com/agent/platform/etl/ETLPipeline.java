package com.agent.platform.etl;

import com.agent.platform.infrastructure.vectordb.MilvusService;
import com.agent.platform.model.dto.DocumentUploadResponse;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.web.multipart.MultipartFile;

import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

/**
 * ETL 数据管道
 * <p>
 * 文档处理完整流程：
 * <ol>
 *   <li>Extract（提取）：使用 DocumentParser 从文件中提取文本</li>
 *   <li>Transform（转换）：使用 DocumentChunker 切分文本为 Chunk</li>
 *   <li>Load（加载）：向量化后存入 Milvus</li>
 * </ol>
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class ETLPipeline {

    private final DocumentParser documentParser;
    private final DocumentChunker documentChunker;
    private final MilvusService milvusService;

    /**
     * 执行完整的 ETL 流程
     *
     * @param file 上传的文件
     * @return 处理结果
     */
    public DocumentUploadResponse process(MultipartFile file) {
        String documentId = UUID.randomUUID().toString().replace("-", "");
        String fileName = file.getOriginalFilename();

        log.info("ETL 管道启动: documentId={}, fileName={}", documentId, fileName);

        // E: 提取文本
        String text = documentParser.parse(file);
        if (text.isBlank()) {
            throw new RuntimeException("文档内容为空，无法处理");
        }

        // T: 切分为 Chunk
        List<DocumentChunker.Chunk> chunks = documentChunker.chunk(text, documentId);
        if (chunks.isEmpty()) {
            throw new RuntimeException("文档切片结果为空");
        }

        // L: 向量化并存入 Milvus
        loadToVectorDB(chunks);

        log.info("ETL 管道完成: documentId={}, chunks={}", documentId, chunks.size());

        return DocumentUploadResponse.builder()
                .documentId(documentId)
                .fileName(fileName)
                .chunkCount(chunks.size())
                .status("completed")
                .build();
    }

    /**
     * 将 Chunk 向量化并存入 Milvus
     * <p>
     * 生产环境应通过 Embedding 模型生成真实向量，
     * 此处使用占位向量演示流程。
     */
    private void loadToVectorDB(List<DocumentChunker.Chunk> chunks) {
        List<String> ids = new ArrayList<>();
        List<List<Float>> vectors = new ArrayList<>();
        List<String> contents = new ArrayList<>();

        for (DocumentChunker.Chunk chunk : chunks) {
            ids.add(chunk.getId());
            vectors.add(generatePlaceholderEmbedding());
            contents.add(chunk.getContent());
        }

        try {
            milvusService.insertVectors(ids, vectors, contents);
            log.info("成功写入 {} 条向量到 Milvus", chunks.size());
        } catch (Exception e) {
            log.error("向量写入 Milvus 失败", e);
            throw new RuntimeException("向量存储失败: " + e.getMessage(), e);
        }
    }

    /**
     * 占位向量生成（1536维，模拟 OpenAI text-embedding-ada-002）
     * 生产环境应替换为真实 Embedding 模型调用
     */
    private List<Float> generatePlaceholderEmbedding() {
        List<Float> embedding = new ArrayList<>(1536);
        for (int i = 0; i < 1536; i++) {
            embedding.add((float) Math.random() * 0.1f);
        }
        return embedding;
    }
}
