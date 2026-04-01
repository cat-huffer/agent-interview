package com.agent.platform.model.entity;

import com.agent.platform.model.enums.MessageRole;
import com.baomidou.mybatisplus.annotation.*;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;

/**
 * 消息实体
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@TableName("message")
public class Message {

    @TableId(type = IdType.ASSIGN_UUID)
    private String id;

    /** 所属会话 ID */
    private String conversationId;

    /** 消息角色 */
    private MessageRole role;

    /** 消息内容 */
    private String content;

    /** Token 数量 */
    private Integer tokenCount;

    /** 使用的模型 */
    private String model;

    /** 逻辑删除 */
    @TableLogic
    private Integer deleted;

    @TableField(fill = FieldFill.INSERT)
    private LocalDateTime createTime;
}
