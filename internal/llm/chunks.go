package llm

import (
	"errors"
	"regexp"
	"strings"
)

// ChunkOffsets represents the offsets for a chunk in the full article.
// |-------------------------- size --------------------------|
// |--------------| 0.5 overlap                |--------------| 0.5 overlap
// Start          OffsetLeft                   OffsetRight    Stop
type ChunkOffsets struct {
	ID          int32 // ID of the chunk, if applicable
	Start       int32 // start index of the chunk in the full text
	OffsetLeft  int32 // start index of the unique content in the chunk
	OffsetRight int32 // end index of the unique content in the chunk
	End         int32 // end index of the chunk in the full text
}

// ChunckOffsets splits a single text into chunks and returns offsets for each chunk in
// the text.
func ChunckOffsets(text string, size, overlap int) ([]ChunkOffsets, error) {
	if size <= 0 {
		return nil, ErrChunkSizeTooSmall
	}
	if overlap <= 1 || overlap >= size || overlap%2 != 0 {
		return nil, ErrInvalidChunkOverlap
	}
	var offsets []ChunkOffsets
	runes := []rune(text)
	textLen := len(runes)
	step := size - overlap
	for i := 0; i < textLen; i += step {
		start := max(0, i-overlap/2)
		end := min(textLen, i+size-overlap/2)
		uniqueStart := i
		uniqueEnd := min(textLen, i+size-overlap)
		offsetLeft := uniqueStart - start
		offsetRight := uniqueEnd - start
		offsets = append(offsets, ChunkOffsets{
			Start:       int32(start),
			OffsetLeft:  int32(offsetLeft),
			OffsetRight: int32(offsetRight),
			End:         int32(end),
		})
		if uniqueEnd >= textLen {
			break
		}
	}
	return offsets, nil
}

// ChunckParagraphsOffsets splits paragraphs into chunks and returns offsets for each chunk in the full article.
func ChunckParagraphsOffsets(paragraphs []string, size, overlap int) ([]ChunkOffsets, error) {
	if size <= 0 {
		return nil, ErrChunkSizeTooSmall
	}
	if overlap <= 1 || overlap >= size || overlap%2 != 0 {
		return nil, ErrInvalidChunkOverlap
	}
	var offsets []ChunkOffsets
	var paraStarts []int
	idx := 0
	for _, para := range paragraphs {
		paraStarts = append(paraStarts, idx)
		idx += len([]rune(para))
	}
	for pi, para := range paragraphs {
		paraRunes := []rune(para)
		paraLen := len(paraRunes)
		paraStart := paraStarts[pi]
		if paraLen == 0 {
			continue
		}
		step := size - overlap
		for i := 0; i < paraLen; i += step {
			startInPara := max(0, i-overlap/2)
			endInPara := min(paraLen, i+size-overlap/2)
			uniqueStartInPara := i
			uniqueEndInPara := min(paraLen, i+size-overlap)
			start := paraStart + startInPara
			end := paraStart + endInPara
			offsetLeft := uniqueStartInPara - startInPara
			offsetRight := uniqueEndInPara - startInPara
			offsets = append(offsets, ChunkOffsets{
				Start:       int32(start),
				OffsetLeft:  int32(offsetLeft),
				OffsetRight: int32(offsetRight),
				End:         int32(end),
			})
			if uniqueEndInPara >= paraLen {
				break
			}
		}
	}
	return offsets, nil
}

// ExtractChunk extracts the chunk, unique content, and overlaps from the article using offsets.
func ExtractChunk(article string, offsets ChunkOffsets) (chunk, leftOverlap, unique, rightOverlap string) {
	runes := []rune(article)
	chunk = string(runes[offsets.Start:offsets.End])
	if offsets.OffsetLeft > 0 {
		leftOverlap = string(runes[offsets.Start : offsets.Start+offsets.OffsetLeft])
	}
	unique = string(runes[offsets.Start+offsets.OffsetLeft : offsets.Start+offsets.OffsetRight])
	if offsets.OffsetRight < offsets.End-offsets.Start {
		rightOverlap = string(runes[offsets.Start+offsets.OffsetRight : offsets.End])
	}
	return
}

// LlmInjectionPatterns contains regex patterns to detect potential LLM injection attacks.
// These patterns are used to identify malicious content that could manipulate the behavior of LLMs.
// The patterns include SQL injection, XSS, llm-specific injections, and other common attack vectors.
// The patterns are designed to be used with regex matching functions.
var LlmInjectionPatterns = []string{
	// SQL Injection patterns
	`(?i)\b(SELECT|INSERT|UPDATE|DELETE|DROP|ALTER|CREATE|EXEC)\b.*?;?`,
	`(?i)\b(OR|AND)\b.*?=\s*['"]?\w+['"]?`,
	`(?i)\bUNION\b.*?\bSELECT\b.*?;?`,
	`(?i)\bEXEC\b.*?\bSP_.*?\b;?`,

	// XSS patterns
	`(?i)<script.*?>.*?</script>`,
	`(?i)<.*?on\w+\s*=\s*['"]?[^'"]*['"]?`,
	`(?i)<.*?javascript:.*?>`,

	// Command injection patterns
	`(?i)\b(system|exec|shell_exec|passthru|popen|proc_open)\b.*?\(`,
	`(?i)\b(cmd|bash|sh)\b.*?[-|;|&]`,

	// LLM-specific injection patterns
	`(?i)ignore\s+(all\s+)?(previous|prior)\s+instructions`,
	`(?i)forget\s+all\s+prior\s+context`,
	`(?i)you\s+are\s+now\s+a?[\w\s]*`,
	`(?i)system:\s*.*`,
	`(?i)user:\s*.*`,
	`(?i)as\s+an\s+ai\s+developed\s+by\s+openai`,
	`(?i)repeat\s+everything\s+i\s+say`,
	`(?i)repeat\s+the\s+prompt\s`,
	`(?i)respond\s+as\s+if\s+you\s+are\s+the\s+system`,
	`(?i)---\s*end\s+of\s+user\s+input\s*---`,
	`(?i)#{3,}\s*new\s+instructions\s*#{3,}`,
	`(?i)"""[\s\S]*?"""`,
	`(?i)'''[\s\S]*?'''`,
	"(?i)```[\\s\\S]*?```",
}

// DetectLlmInjection checks if the input string contains patterns that indicate potential
// LLM injection attacks. It returns true if any of the patterns match, indicating a
// potential injection attempt.
func DetectLlmInjection(input string) (bool, string) {
	for _, pattern := range LlmInjectionPatterns {
		if matched, _ := regexp.MatchString(pattern, input); matched {
			return true, pattern
		}
	}
	return false, ""
}

// chunk represents a text chunk with three parts: left overlap, main content, and right overlap.
type chunk [3]string

// String returns a string representation of the chunk, joining the three parts with a separator.
func (c chunk) String() string {
	return strings.Join(c[:], " | ")
}

var ErrChunkSizeTooSmall = errors.New("chunk size must be greater than 0")
var ErrInvalidChunkOverlap = errors.New("chunk overlap must be an even number greater than 1 and less than chunk size")

// Chunck splits the input text into chunks of a specified size with a defined
// overlap. Overlap should be an even number that is less than the chunk size.
func Chunck(text string, size int, overlap int) ([]chunk, error) {
	//  |------------- size -------------|
	//  |-----|    0.5 overlap     |-----| 0.5 overlap
	//  | l_o |        l_u		   | l_o |
	if size <= 0 {
		return nil, ErrChunkSizeTooSmall
	}

	if overlap <= 1 || overlap >= size || overlap%2 != 0 {
		return nil, ErrInvalidChunkOverlap
	}

	var chunks []chunk
	lo, lu := overlap/2, size-overlap

	runes := []rune(text)
	lhs, rhs := 0, min(size-2*lo, len(runes))
	for {
		c := chunk{
			string(runes[max(lhs-lo, 0):lhs]),
			string(runes[lhs:min(rhs, len(runes))]),
			string(runes[rhs:min(rhs+lo, len(runes))]),
		}
		chunks = append(chunks, c)
		if rhs >= len(runes) {
			break
		}
		lhs += lu
		rhs = min(rhs+lu, len(runes))
	}
	return chunks, nil
}

// ChunckParagraphs splits paragraphs into chunks of a specified size with a defined
// overlap. Each paragraph is treated as a separate entity, and the function ensures
// that chunks are created with the specified overlap. The function handles paragraphs
// that are shorter than the chunk size by including context from adjacent paragraphs.
func ChunckParagraphs(paragraphs []string, size int, overlap int) ([]chunk, error) {
	if size <= 0 {
		return nil, ErrChunkSizeTooSmall
	}

	if overlap <= 1 || overlap >= size || overlap%2 != 0 {
		return nil, ErrInvalidChunkOverlap
	}

	var chunks []chunk
	lo, lu := overlap/2, size-overlap

	runes := make([][]rune, len(paragraphs))
	for i, p := range paragraphs {
		runes[i] = []rune(p)
	}

	for i, rs := range runes {
		if len(rs) == 0 {
			continue
		}

		if len(rs) <= lu {
			c := chunk{"", string(rs), ""}
			if i > 0 {
				c[0] = string(runes[i-1][max(0, len(runes[i-1])-lo):])
			}
			if i < len(runes)-1 {
				c[2] = string(runes[i+1][:min(lo, len(runes[i+1]))])
			}
			chunks = append(chunks, c)
		} else {
			cs, err := Chunck(string(rs), size, overlap)
			if err != nil {
				return nil, err
			}

			if i > 1 {
				cs[0][0] = string(runes[max(0, i-1)][max(0, len(runes[max(0, i-1)])-lo):])
			}

			if i < len(paragraphs)-1 {
				cs[len(cs)-1][2] = string(runes[i+1][:min(lo, len(runes[i+1]))])
			}
			chunks = append(chunks, cs...)
		}
	}
	return chunks, nil
}
