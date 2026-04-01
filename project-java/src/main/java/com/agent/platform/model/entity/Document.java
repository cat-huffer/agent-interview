package com.agent.platform.model.entity;

import com.baomidou.mybatisplus.annotation.*;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;

/**
 * 文档实体
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@TableName("document")
public class Document {

    @TableId(type = IdType.ASSIGN_UUID)
    private String id;

    /** 文件名 */
    private String fileName;

    /** 文件类型（pdf, docx, txt 等） */
    private String fileType;

    /** 文件大小（字节） */
    private Long fileSize;

    /** 文件存储路径 */
    private String filePath;

    /** 切片数量 */
    private Integer chunkCount;

    /** 处理状态：pending / processing / completed / failed */
    private String status;

    /** 上传用户 */
    private String userId;

    @TableLogic
    private Integer deleted;

    @TableField(fill = FieldFill.INSERT)
    private LocalDateTime createTime;

    @TableField(fill = FieldFill.INSERT_UPDATE)
    private LocalDateTime updateTime;
}
