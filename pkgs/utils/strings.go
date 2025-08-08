package utils

import (
	"regexp"
	"strings"
)

// should rename utils package into a more specific name if the package grows
// since it is too generic

func NormalizeString(s string) string {
	s = ReplaceNonBreakingSpaces(s)
	// s = ConvertFullwidthToASCII(s)
	s = RemoveSpace(s)
	s = RemoveInvisibleChars(s)
	return s
}

func RemoveInvisibleChars(s string) string {
	// remove invisible characters from the string
	re := regexp.MustCompile(`[\x00-\x1F\x7F-\x9F　 ]`)
	return re.ReplaceAllString(s, "")
}

func ReplaceNonBreakingSpaces(s string) string {
	// replace non-breaking spaces with regular spaces
	// this regex will match any non-breaking space character
	return strings.ReplaceAll(s, "\u00A0", " ")
}

func RemoveSpace(s string) string {
	// remove extra spaces from the string
	// this regex will match any whitespace character (space, tab, newline, etc.)
	// and replace it with a single space
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, " ")
}

func ConvertFullwidthToASCII(s string) string {
	fullwidth := map[rune]rune{
		'！': '!',
		'？': '?',
		'，': ',',
		'。': '｡',
		'：': ':',
		'；': ';',
		'（': '(',
		'）': ')',
		'［': '[',
		'］': ']',
		'｛': '{',
		'｝': '}',
		'【': '[',
		'】': ']',
		'「': '"',
		'」': '"',
		'『': '"',
		'』': '"',
		'、': ',',
		'《': '<',
		'》': '>',
		'〈': '<',
		'〉': '>',
		'～': '~',
		'＃': '#',
		'％': '%',
		'＆': '&',
		'＊': '*',
		'＠': '@',
		'＄': '$',
		'＾': '^',
		'＿': '_',
		'＋': '+',
		'－': '-',
		'＝': '=',
		'｜': '|',
		'＼': '\\',
		'／': '/',
		'＂': '"',
		'＇': '\'',
		'｀': '`',
		'＜': '<',
		'＞': '>',
	}
	result := []rune{}
	for _, r := range s {
		if ascii, ok := fullwidth[r]; ok {
			result = append(result, ascii)
		} else if r >= '！' && r <= '～' {
			// Convert fullwidth ASCII range to normal ASCII
			result = append(result, r-0xFEE0)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func Join(text []string) (string, []int) {
	builder := strings.Builder{}
	cuts := make([]int, 0, len(text))
	for _, t := range text {
		builder.WriteString(t)
		cuts = append(cuts, builder.Len())
	}
	return builder.String(), cuts
}

func Split(text string, cuts []int) []string {
	if len(cuts) == 0 {
		return []string{text}
	}
	result := make([]string, 0, len(cuts))
	head := 0
	for _, tail := range cuts {
		if tail > head {
			result = append(result, text[head:tail])
		}
		head = tail
	}
	return result
}

func Mask(pwd string) string {
	if len(pwd) <= 10 {
		return strings.Repeat("●", len(pwd))
	}
	return pwd[:5] + strings.Repeat("●", min(len(pwd)-10, 10)) + pwd[len(pwd)-5:]
}
