package etl

import (
	"strings"
	"unicode/utf8"
)

// ChunkStrategy 分块策略
type ChunkStrategy int

const (
	// StrategyFixedSize 固定大小分块
	StrategyFixedSize ChunkStrategy = iota
	// StrategySentence 按句子分块
	StrategySentence
	// StrategySemantic 按语义段落分块
	StrategySemantic
)

// Chunk 文档分块
type Chunk struct {
	Content    string
	ChunkIndex int
	StartPos   int
	EndPos     int
}

// Chunker 文档分块器
type Chunker struct {
	chunkSize int // 每块的最大字符数
	overlap   int // 相邻块的重叠字符数
}

// NewChunker 创建分块器
func NewChunker(chunkSize, overlap int) *Chunker {
	if chunkSize <= 0 {
		chunkSize = 512
	}
	if overlap < 0 || overlap >= chunkSize {
		overlap = chunkSize / 8
	}
	return &Chunker{
		chunkSize: chunkSize,
		overlap:   overlap,
	}
}

// Split 将文档内容分割为多个块
func (c *Chunker) Split(content string, strategy ChunkStrategy) []Chunk {
	switch strategy {
	case StrategySentence:
		return c.splitBySentence(content)
	case StrategySemantic:
		return c.splitBySemantic(content)
	default:
		return c.splitByFixedSize(content)
	}
}

// splitByFixedSize 按固定字符数分块，支持滑动窗口重叠
func (c *Chunker) splitByFixedSize(content string) []Chunk {
	runes := []rune(content)
	totalLen := len(runes)
	if totalLen == 0 {
		return nil
	}

	var chunks []Chunk
	step := c.chunkSize - c.overlap
	if step <= 0 {
		step = c.chunkSize
	}

	for i := 0; i < totalLen; i += step {
		end := i + c.chunkSize
		if end > totalLen {
			end = totalLen
		}

		chunkContent := string(runes[i:end])
		chunks = append(chunks, Chunk{
			Content:    strings.TrimSpace(chunkContent),
			ChunkIndex: len(chunks),
			StartPos:   i,
			EndPos:     end,
		})

		if end >= totalLen {
			break
		}
	}

	return chunks
}

// splitBySentence 按句子分块，尽量在句子边界处切分
func (c *Chunker) splitBySentence(content string) []Chunk {
	// 按句子分隔符切分
	separators := []string{"。", "！", "？", ".", "!", "?", "\n\n"}
	sentences := splitBySeparators(content, separators)

	var chunks []Chunk
	var current strings.Builder
	currentLen := 0

	for _, sentence := range sentences {
		sentLen := utf8.RuneCountInString(sentence)

		if currentLen+sentLen > c.chunkSize && currentLen > 0 {
			// 当前块已满，保存并开始新块
			chunks = append(chunks, Chunk{
				Content:    strings.TrimSpace(current.String()),
				ChunkIndex: len(chunks),
			})

			// 重叠处理：保留最后一部分
			overlapText := getLastNChars(current.String(), c.overlap)
			current.Reset()
			current.WriteString(overlapText)
			currentLen = utf8.RuneCountInString(overlapText)
		}

		current.WriteString(sentence)
		currentLen += sentLen
	}

	if currentLen > 0 {
		chunks = append(chunks, Chunk{
			Content:    strings.TrimSpace(current.String()),
			ChunkIndex: len(chunks),
		})
	}

	return chunks
}

// splitBySemantic 按语义段落分块（基于段落标题和空行）
func (c *Chunker) splitBySemantic(content string) []Chunk {
	// 按双换行符分割段落
	paragraphs := strings.Split(content, "\n\n")

	var chunks []Chunk
	var current strings.Builder
	currentLen := 0

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		paraLen := utf8.RuneCountInString(para)

		if currentLen+paraLen > c.chunkSize && currentLen > 0 {
			chunks = append(chunks, Chunk{
				Content:    strings.TrimSpace(current.String()),
				ChunkIndex: len(chunks),
			})
			current.Reset()
			currentLen = 0
		}

		current.WriteString(para + "\n\n")
		currentLen += paraLen
	}

	if currentLen > 0 {
		chunks = append(chunks, Chunk{
			Content:    strings.TrimSpace(current.String()),
			ChunkIndex: len(chunks),
		})
	}

	return chunks
}

// splitBySeparators 按多个分隔符切分文本，保留分隔符
func splitBySeparators(text string, separators []string) []string {
	var parts []string
	remaining := text

	for len(remaining) > 0 {
		minIdx := len(remaining)
		minSep := ""

		for _, sep := range separators {
			idx := strings.Index(remaining, sep)
			if idx >= 0 && idx < minIdx {
				minIdx = idx
				minSep = sep
			}
		}

		if minSep == "" {
			parts = append(parts, remaining)
			break
		}

		part := remaining[:minIdx+len(minSep)]
		if strings.TrimSpace(part) != "" {
			parts = append(parts, part)
		}
		remaining = remaining[minIdx+len(minSep):]
	}

	return parts
}

// getLastNChars 获取字符串最后 N 个字符
func getLastNChars(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[len(runes)-n:])
}
