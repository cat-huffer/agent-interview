package com.agent.platform.service.tool.builtin;

import com.agent.platform.service.tool.BaseTool;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

import javax.script.ScriptEngine;
import javax.script.ScriptEngineManager;
import java.util.Map;

/**
 * 计算器工具
 * <p>
 * 支持数学表达式计算，使用 JavaScript 引擎求值
 */
@Slf4j
@Component
public class CalculatorTool extends BaseTool {

    @Override
    public String getName() {
        return "calculator";
    }

    @Override
    public String getDescription() {
        return "数学计算工具，可以计算数学表达式。支持加减乘除、幂运算、三角函数等。当需要精确计算时使用。";
    }

    @Override
    public String getParameterSchema() {
        return """
                {
                  "type": "object",
                  "properties": {
                    "expression": {
                      "type": "string",
                      "description": "数学表达式，如 '2 + 3 * 4' 或 'Math.sqrt(16)'"
                    }
                  },
                  "required": ["expression"]
                }
                """;
    }

    @Override
    protected void validate(Map<String, Object> parameters) {
        if (!parameters.containsKey("expression") || parameters.get("expression").toString().isBlank()) {
            throw new IllegalArgumentException("数学表达式不能为空");
        }
    }

    @Override
    protected String doExecute(Map<String, Object> parameters) {
        String expression = parameters.get("expression").toString();
        log.info("执行计算: expression={}", expression);

        try {
            ScriptEngineManager manager = new ScriptEngineManager();
            ScriptEngine engine = manager.getEngineByName("js");

            if (engine == null) {
                return evaluateBasicExpression(expression);
            }

            Object result = engine.eval(expression);
            return "计算结果: " + expression + " = " + result;
        } catch (Exception e) {
            return evaluateBasicExpression(expression);
        }
    }

    /**
     * 基础四则运算备用方案
     */
    private String evaluateBasicExpression(String expression) {
        try {
            String cleaned = expression.replaceAll("[^0-9+\\-*/().\\s]", "");
            double result = new Object() {
                int pos = -1, ch;

                void nextChar() {
                    ch = (++pos < cleaned.length()) ? cleaned.charAt(pos) : -1;
                }

                boolean eat(int charToEat) {
                    while (ch == ' ') nextChar();
                    if (ch == charToEat) {
                        nextChar();
                        return true;
                    }
                    return false;
                }

                double parse() {
                    nextChar();
                    double x = parseExpression();
                    return x;
                }

                double parseExpression() {
                    double x = parseTerm();
                    for (;;) {
                        if (eat('+')) x += parseTerm();
                        else if (eat('-')) x -= parseTerm();
                        else return x;
                    }
                }

                double parseTerm() {
                    double x = parseFactor();
                    for (;;) {
                        if (eat('*')) x *= parseFactor();
                        else if (eat('/')) x /= parseFactor();
                        else return x;
                    }
                }

                double parseFactor() {
                    if (eat('+')) return +parseFactor();
                    if (eat('-')) return -parseFactor();

                    double x;
                    int startPos = this.pos;
                    if (eat('(')) {
                        x = parseExpression();
                        eat(')');
                    } else if ((ch >= '0' && ch <= '9') || ch == '.') {
                        while ((ch >= '0' && ch <= '9') || ch == '.') nextChar();
                        x = Double.parseDouble(cleaned.substring(startPos, this.pos));
                    } else {
                        throw new RuntimeException("无法解析的字符: " + (char) ch);
                    }

                    return x;
                }
            }.parse();

            return "计算结果: " + expression + " = " + result;
        } catch (Exception e) {
            return "计算失败: 无法解析表达式 '" + expression + "'";
        }
    }
}
