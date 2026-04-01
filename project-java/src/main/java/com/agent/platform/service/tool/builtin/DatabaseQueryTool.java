package com.agent.platform.service.tool.builtin;

import com.agent.platform.service.tool.BaseTool;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

/**
 * 数据库查询工具
 * <p>
 * 仅允许 SELECT 查询操作，严禁写操作，防止 SQL 注入风险
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class DatabaseQueryTool extends BaseTool {

    private final JdbcTemplate jdbcTemplate;

    private static final int MAX_ROWS = 100;
    private static final Set<String> FORBIDDEN_KEYWORDS = Set.of(
            "INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE",
            "TRUNCATE", "GRANT", "REVOKE", "EXEC", "EXECUTE"
    );

    @Override
    public String getName() {
        return "database_query";
    }

    @Override
    public String getDescription() {
        return "执行 SQL 查询语句（仅支持 SELECT），返回查询结果。用于需要从数据库获取数据时使用。";
    }

    @Override
    public String getParameterSchema() {
        return """
                {
                  "type": "object",
                  "properties": {
                    "sql": {
                      "type": "string",
                      "description": "SQL 查询语句（仅支持 SELECT）"
                    }
                  },
                  "required": ["sql"]
                }
                """;
    }

    @Override
    protected void validate(Map<String, Object> parameters) {
        String sql = parameters.getOrDefault("sql", "").toString().trim();
        if (sql.isBlank()) {
            throw new IllegalArgumentException("SQL 语句不能为空");
        }

        String upperSql = sql.toUpperCase();
        for (String keyword : FORBIDDEN_KEYWORDS) {
            if (upperSql.contains(keyword)) {
                throw new IllegalArgumentException("安全限制：禁止执行写操作，仅允许 SELECT 查询");
            }
        }

        if (!upperSql.startsWith("SELECT")) {
            throw new IllegalArgumentException("仅支持 SELECT 查询语句");
        }
    }

    @Override
    protected String doExecute(Map<String, Object> parameters) {
        String sql = parameters.get("sql").toString().trim();

        if (!sql.toUpperCase().contains("LIMIT")) {
            sql = sql.replaceAll(";\\s*$", "") + " LIMIT " + MAX_ROWS;
        }

        log.info("执行 SQL 查询: {}", sql);

        List<Map<String, Object>> results = jdbcTemplate.queryForList(sql);

        if (results.isEmpty()) {
            return "查询结果为空";
        }

        StringBuilder sb = new StringBuilder();
        sb.append("查询结果（共 ").append(results.size()).append(" 行）:\n\n");

        String header = String.join(" | ", results.get(0).keySet());
        sb.append(header).append("\n");
        sb.append("-".repeat(header.length())).append("\n");

        for (Map<String, Object> row : results) {
            String rowStr = row.values().stream()
                    .map(v -> v == null ? "NULL" : v.toString())
                    .collect(Collectors.joining(" | "));
            sb.append(rowStr).append("\n");
        }

        return sb.toString();
    }
}
