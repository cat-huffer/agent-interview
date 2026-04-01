package com.agent.platform.etl;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

/**
 * 文档切片器
 * <p>
 * 将长文本按照语义边界切分为小块（Chunk），
 * 支持固定大小切片和段落感知切片，带有重叠以保持上下文连续性。
 */
@Slf4j
@Component
public class DocumentChunker {

    @Value("${agent.rag.chunk-size:512}")
    private int chunkSize;

    @Value("${agent.rag.chunk-overlap:64}")
    private int chunkOverlap;

    /**
     * 将文本切分为 Chunk 列表
     *
     * @param text       原始文本
     * @param documentId 文档 ID
     * @return Chunk 列表
     */
    public List<Chunk> chunk(String text, String documentId) {
        if (text == null || text.isBlank()) {
            return List.of();
        }

        List<Chunk> chunks = new ArrayList<>();

        String[] paragraphs = text.split("\n\n");
        StringBuilder buffer = new StringBuilder();
        int chunkIndex = 0;

        for (String paragraph : paragraphs) {
            if (buffer.length() + paragraph.length() + 1 > chunkSize && buffer.length() > 0) {
                chunks.add(createChunk(buffer.toString(), documentId, chunkIndex++));

                int overlapStart = Math.max(0, buffer.length() - chunkOverlap);
                String overlap = buffer.substring(overlapStart);
                buffer = new StringBuilder(overlap);
            }

            if (buffer.length() > 0) {
                buffer.append("\n\n");
            }
            buffer.append(paragraph);

            if (paragraph.length() > chunkSize) {
                List<Chunk> subChunks = splitLongParagraph(paragraph, documentId, chunkIndex);
                chunks.addAll(subChunks);
                chunkIndex += subChunks.size();
                buffer = new StringBuilder();
            }
        }

        if (buffer.length() > 0) {
            chunks.add(createChunk(buffer.toString(), documentId, chunkIndex));
        }

        log.info("文档切片完成: documentId={}, totalChunks={}, avgChunkSize={}",
                documentId, chunks.size(),
                chunks.isEmpty() ? 0 : chunks.stream().mapToInt(c -> c.getContent().length()).average().orElse(0));

        return chunks;
    }

    /**
     * 超长段落的强制切分
     */
    private List<Chunk> splitLongParagraph(String paragraph, String documentId, int startIndex) {
        List<Chunk> chunks = new ArrayList<>();
        int index = startIndex;

        for (int i = 0; i < paragraph.length(); i += chunkSize - chunkOverlap) {
            int end = Math.min(i + chunkSize, paragraph.length());
            String segment = paragraph.substring(i, end);
            chunks.add(createChunk(segment, documentId, index++));
        }

        return chunks;
    }

    private Chunk createChunk(String content, String documentId, int index) {
        return Chunk.builder()
                .id(UUID.randomUUID().toString().replace("-", "").substring(0, 16))
                .documentId(documentId)
                .content(content.trim())
                .index(index)
                .charCount(content.trim().length())
                .build();
    }

    @Data
    @Builder
    @AllArgsConstructor
    public static class Chunk {
        private String id;
        private String documentId;
        private String content;
        private int index;
        private int charCount;
    }
}
