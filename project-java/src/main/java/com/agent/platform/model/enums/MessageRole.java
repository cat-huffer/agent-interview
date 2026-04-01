package com.agent.platform.model.enums;

import lombok.AllArgsConstructor;
import lombok.Getter;

/**
 * 消息角色枚举
 */
@Getter
@AllArgsConstructor
public enum MessageRole {

    SYSTEM("system", "系统提示"),
    USER("user", "用户消息"),
    ASSISTANT("assistant", "助手回复"),
    TOOL("tool", "工具调用结果");

    private final String code;
    private final String description;
}
