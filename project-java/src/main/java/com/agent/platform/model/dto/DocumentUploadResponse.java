package com.agent.platform.model.dto;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * 文档上传响应 DTO
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class DocumentUploadResponse {

    /** 文档 ID */
    private String documentId;

    /** 原始文件名 */
    private String fileName;

    /** 文档切片数量 */
    private int chunkCount;

    /** 向量化状态 */
    private String status;

    /** 处理耗时（毫秒） */
    private long processingTimeMs;
}
