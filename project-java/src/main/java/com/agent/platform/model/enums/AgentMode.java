package com.agent.platform.model.enums;

import lombok.AllArgsConstructor;
import lombok.Getter;

/**
 * Agent 运行模式枚举
 */
@Getter
@AllArgsConstructor
public enum AgentMode {

    REACT("react", "ReAct 推理-行动循环模式"),
    PLANNER("planner", "规划-执行分步模式"),
    REFLECTION("reflection", "自我反思改进模式"),
    DIRECT("direct", "直接对话模式，不走 Agent 编排");

    private final String code;
    private final String description;

    public static AgentMode fromCode(String code) {
        for (AgentMode mode : values()) {
            if (mode.code.equalsIgnoreCase(code)) {
                return mode;
            }
        }
        return REACT;
    }
}
