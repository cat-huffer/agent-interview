package com.agent.platform.controller;

import com.agent.platform.common.Result;
import com.agent.platform.model.dto.DocumentUploadResponse;
import com.agent.platform.etl.ETLPipeline;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;

/**
 * 文档管理控制器
 * <p>
 * 提供文档上传、解析、向量化的入口
 */
@Slf4j
@RestController
@RequestMapping("/api/v1/documents")
@RequiredArgsConstructor
public class DocumentController {

    private final ETLPipeline etlPipeline;

    /**
     * 上传并处理文档
     * <p>
     * 支持 PDF、DOCX、TXT、MD 等格式。
     * 上传后自动执行：解析 → 切片 → 向量化 → 存入 Milvus
     */
    @PostMapping("/upload")
    public Result<DocumentUploadResponse> upload(@RequestParam("file") MultipartFile file) {
        if (file.isEmpty()) {
            return Result.fail(400, "上传文件不能为空");
        }

        String fileName = file.getOriginalFilename();
        log.info("收到文档上传请求: fileName={}, size={}KB",
                fileName, file.getSize() / 1024);

        long startTime = System.currentTimeMillis();

        try {
            DocumentUploadResponse response = etlPipeline.process(file);
            response.setProcessingTimeMs(System.currentTimeMillis() - startTime);

            log.info("文档处理完成: fileName={}, chunks={}, elapsed={}ms",
                    fileName, response.getChunkCount(), response.getProcessingTimeMs());

            return Result.success(response);
        } catch (Exception e) {
            log.error("文档处理失败: fileName={}", fileName, e);
            return Result.fail("文档处理失败: " + e.getMessage());
        }
    }
}
