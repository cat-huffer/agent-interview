// Package etl 提供文档的 ETL（Extract-Transform-Load）处理流水线。
// 负责文档解析、分块、向量化和入库的完整流程。
package etl

import (
	"context"
	"fmt"
	"strings"
)

// DocumentType 文档类型
type DocumentType string

const (
	DocTypeText     DocumentType = "text"
	DocTypeMarkdown DocumentType = "markdown"
	DocTypeHTML     DocumentType = "html"
	DocTypePDF      DocumentType = "pdf"
)

// ParsedDocument 解析后的文档
type ParsedDocument struct {
	Title    string
	Content  string
	Sections []Section
	Metadata map[string]string
}

// Section 文档中的一个章节
type Section struct {
	Title   string
	Content string
	Level   int // 标题级别
}

// Parser 文档解析器接口
type Parser interface {
	// Parse 将原始文档内容解析为结构化格式
	Parse(ctx context.Context, content string, docType DocumentType) (*ParsedDocument, error)
}

// DefaultParser 默认文档解析器
type DefaultParser struct{}

// NewDefaultParser 创建默认解析器
func NewDefaultParser() *DefaultParser {
	return &DefaultParser{}
}

// Parse 解析文档内容
func (p *DefaultParser) Parse(ctx context.Context, content string, docType DocumentType) (*ParsedDocument, error) {
	switch docType {
	case DocTypeMarkdown:
		return p.parseMarkdown(content)
	case DocTypeHTML:
		return p.parseHTML(content)
	case DocTypeText:
		return p.parseText(content)
	default:
		return p.parseText(content)
	}
}

// parseMarkdown 解析 Markdown 文档，提取标题层级结构
func (p *DefaultParser) parseMarkdown(content string) (*ParsedDocument, error) {
	doc := &ParsedDocument{
		Content:  content,
		Metadata: make(map[string]string),
	}

	lines := strings.Split(content, "\n")
	var currentSection *Section

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 检测标题
		if strings.HasPrefix(trimmed, "#") {
			level := 0
			for _, ch := range trimmed {
				if ch == '#' {
					level++
				} else {
					break
				}
			}
			title := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))

			if doc.Title == "" && level == 1 {
				doc.Title = title
			}

			if currentSection != nil {
				doc.Sections = append(doc.Sections, *currentSection)
			}
			currentSection = &Section{Title: title, Level: level}
		} else if currentSection != nil {
			currentSection.Content += line + "\n"
		}
	}

	if currentSection != nil {
		doc.Sections = append(doc.Sections, *currentSection)
	}

	return doc, nil
}

// parseHTML 解析 HTML 文档（简化实现，实际应使用 goquery 等库）
func (p *DefaultParser) parseHTML(content string) (*ParsedDocument, error) {
	// 实际项目中应使用 goquery 进行 HTML 解析
	// 此处进行简单的标签清理
	cleaned := content
	// 移除 HTML 标签（简化处理）
	for strings.Contains(cleaned, "<") {
		start := strings.Index(cleaned, "<")
		end := strings.Index(cleaned, ">")
		if end > start {
			cleaned = cleaned[:start] + cleaned[end+1:]
		} else {
			break
		}
	}

	return &ParsedDocument{
		Content:  strings.TrimSpace(cleaned),
		Metadata: make(map[string]string),
	}, nil
}

// parseText 解析纯文本
func (p *DefaultParser) parseText(content string) (*ParsedDocument, error) {
	if content == "" {
		return nil, fmt.Errorf("文档内容为空")
	}

	// 取第一行作为标题
	lines := strings.SplitN(content, "\n", 2)
	title := strings.TrimSpace(lines[0])

	return &ParsedDocument{
		Title:    title,
		Content:  content,
		Metadata: make(map[string]string),
	}, nil
}
