package agent

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// ── BERT WordPiece 分词器 ─────────────────────────
//
// 支持 bge-small-zh-v1.5 等 BERT 系列模型的分词。
// 加载 .pair/embeddings/<模型>/vocab.txt 使用。
// 纯 Go 实现，零外部依赖。

const (
	bertMaxSeqLen = 512
	bertCLS       = "[CLS]"
	bertSEP       = "[SEP]"
	bertPAD       = "[PAD]"
	bertUNK       = "[UNK]"
	bertMASK      = "[MASK]"

	clsID  = 101
	sepID  = 102
	padID  = 0
	unkID  = 100
	maskID = 103
)

// BERTTokenizer BERT WordPiece 分词器。
type BERTTokenizer struct {
	vocab    map[string]int // token → id
	ids      []string       // id → token（用于调试）
	vocabDir string         // 词表文件所在目录
}

// LoadBERTTokenizer 从指定目录加载 vocab.txt。
// 目录结构：.pair/embeddings/<模型名>/vocab.txt
func LoadBERTTokenizer(vocabDir string) (*BERTTokenizer, error) {
	vocabPath := filepath.Join(vocabDir, "vocab.txt")
	f, err := os.Open(vocabPath)
	if err != nil {
		return nil, fmt.Errorf("无法打开词表 %s: %w", vocabPath, err)
	}
	defer f.Close()

	t := &BERTTokenizer{
		vocab:    make(map[string]int),
		ids:      make([]string, 0),
		vocabDir: vocabDir,
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		token := strings.TrimSpace(scanner.Text())
		id := len(t.ids)
		t.vocab[token] = id
		t.ids = append(t.ids, token)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取词表失败: %w", err)
	}
	if len(t.ids) == 0 {
		return nil, fmt.Errorf("词表为空: %s", vocabPath)
	}
	return t, nil
}

// ID 返回 token 的 ID，未知 token 返回 unkID。
func (t *BERTTokenizer) ID(token string) int {
	if id, ok := t.vocab[token]; ok {
		return id
	}
	if id, ok := t.vocab[strings.ToLower(token)]; ok {
		return id
	}
	return unkID
}

// Encode 将文本编码为 BERT 输入（input_ids + attention_mask）。
// 返回：(input_ids, attention_mask, err)
// 自动添加 [CLS] 前缀和 [SEP] 后缀，截断至 maxSeqLen。
func (t *BERTTokenizer) Encode(text string) ([]int64, []int64, error) {
	tokens := t.tokenize(text)

	// 截断
	maxTokens := bertMaxSeqLen - 2 // 留 [CLS] 和 [SEP]
	if len(tokens) > maxTokens {
		tokens = tokens[:maxTokens]
	}

	// 拼装： [CLS] tokens... [SEP]
	ids := make([]int64, 0, len(tokens)+2)
	ids = append(ids, int64(clsID))
	for _, tok := range tokens {
		ids = append(ids, int64(t.ID(tok)))
	}
	ids = append(ids, int64(sepID))

	// attention_mask
	mask := make([]int64, len(ids))
	for i := range mask {
		mask[i] = 1
	}

	// padding 到 512（BERT 固定输入长度）
	padLen := bertMaxSeqLen - len(ids)
	if padLen > 0 {
		for i := 0; i < padLen; i++ {
			ids = append(ids, int64(padID))
			mask = append(mask, 0)
		}
	}

	return ids, mask, nil
}

// tokenize 对文本做 WordPiece 分词（含中文 CJK 拆分）。
func (t *BERTTokenizer) tokenize(text string) []string {
	// 1. 标准化：小写 + NFKC
	normalized := strings.ToLower(text)

	// 2. 预分词：按 CJK 字符拆分
	var pretokens []string
	var current strings.Builder
	for _, r := range normalized {
		if isCJKRune(r) {
			// 遇到 CJK 字符：flush 当前缓存
			if current.Len() > 0 {
				pretokens = append(pretokens, current.String())
				current.Reset()
			}
			pretokens = append(pretokens, string(r))
		} else if unicode.IsSpace(r) || r == '\t' || r == '\n' || r == '\r' {
			if current.Len() > 0 {
				pretokens = append(pretokens, current.String())
				current.Reset()
			}
		} else if unicode.IsPunct(r) || isASCIIPunct(r) {
			if current.Len() > 0 {
				pretokens = append(pretokens, current.String())
				current.Reset()
			}
			pretokens = append(pretokens, string(r))
		} else {
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		pretokens = append(pretokens, current.String())
	}

	// 3. WordPiece 子词分割
	var tokens []string
	for _, token := range pretokens {
		subTokens := t.wordPiece(token)
		tokens = append(tokens, subTokens...)
	}
	return tokens
}

// wordPiece 对单个词做 WordPiece 子词分割。
func (t *BERTTokenizer) wordPiece(token string) []string {
	if _, ok := t.vocab[token]; ok {
		return []string{token}
	}

	var tokens []string
	remaining := token
	for len(remaining) > 0 {
		// 最长匹配
		substr := remaining
		found := false
		for len(substr) > 0 {
			prefix := substr
			if len(tokens) > 0 {
				prefix = "##" + substr
			}
			if _, ok := t.vocab[prefix]; ok {
				tokens = append(tokens, prefix)
				remaining = remaining[len(substr):]
				found = true
				break
			}
			// 去掉最后一个字符重试
			runes := []rune(substr)
			substr = string(runes[:len(runes)-1])
		}
		if !found {
			// 单个字符仍匹配不上 → 用 [UNK]
			tokens = append(tokens, "[UNK]")
			break
		}
	}
	return tokens
}

// ── 辅助函数 ──────────────────────────────────────

// isCJKRune 判断是否为中日韩表意文字。
func isCJKRune(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r) ||
		(r >= 0x3000 && r <= 0x303F) || // CJK 符号
		(r >= 0xFF00 && r <= 0xFFEF) // 全角字符
}

// isASCIIPunct 判断是否为 ASCII 标点。
func isASCIIPunct(r rune) bool {
	return (r >= 33 && r <= 47) || (r >= 58 && r <= 64) ||
		(r >= 91 && r <= 96) || (r >= 123 && r <= 126)
}

// bertIsSpace 扩展空白判断（含全角空格）。
func bertIsSpace(r rune) bool {
	return unicode.IsSpace(r) || r == 0x3000
}

// ── 待办：完整的 NFKC 标准化（当前用 strings.ToLower 简化）──

// normalizeBERT 对文本做 BERT 标准化（小写 + NFKC）。
func normalizeBERT(text string) string {
	// strings.ToLower 对 ASCII 足够了；中文无需大小写转换
	// 完整的 NFKC 需要 golang.org/x/text 库
	return strings.ToLower(text)
}

// π 常量。
const pi = math.Pi
