package utils

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/pgvector/pgvector-go"
	"golang.org/x/net/idna"
)

// CharSet represents a set of characters as a string.
type CharSet string

func (c CharSet) String() string {
	return string(c)
}

func (c CharSet) Runes() []rune {
	runes := make([]rune, 0, len(c))
	for _, r := range c {
		runes = append(runes, r)
	}
	return runes
}

func (c CharSet) Contains(r rune) bool {
	for _, cr := range c {
		if cr == r {
			return true
		}
	}
	return false
}

func (c CharSet) Len() int {
	return len(c)
}

func MergeCharSets(sets ...CharSet) CharSet {
	set := make(map[rune]struct{})
	for _, s := range sets {
		for _, r := range s {
			set[r] = struct{}{}
		}
	}
	sortedRunes := make([]rune, 0, len(set))
	for r := range set {
		sortedRunes = append(sortedRunes, r)
	}
	sort.Slice(sortedRunes, func(i, j int) bool {
		return sortedRunes[i] < sortedRunes[j]
	})
	return CharSet(string(sortedRunes))
}

const (
	CharSetUpperCase     CharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	CharSetLowerCase     CharSet = "abcdefghijklmnopqrstuvwxyz"
	CharSetNumbers       CharSet = "0123456789"
	CharSetSymbols       CharSet = "!@#$%^&*()-_=+[]{}|;:,.<>?/~`"
	CharSetWhitespace    CharSet = " \t\n\r\f\v"
	CharSetPunctuation   CharSet = ".,;:!?\"'()[]{}<>-=_+`~@#$%^&*|\\/\\"
	CharSetHexDigits     CharSet = "0123456789abcdefABCDEF"
	CharSetBinary        CharSet = "01"
	CharSetOctal         CharSet = "01234567"
	CharSetBase64        CharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	CharSetBase32        CharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	CharSetURLSafeBase64 CharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	CharSetUUID          CharSet = "0123456789abcdefABCDEF-"
	CharSetAlpha         CharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	CharSetAlphaNumeric  CharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	CharSetPrintable     CharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~ "
)

var (
	CharPasswordDefault = MergeCharSets(
		CharSetUpperCase,
		CharSetLowerCase,
		CharSetNumbers,
		CharSetSymbols,
		CharSet(" "),
	)
)

var (
	ErrInvalidLength    = errors.New("invalid length")
	ErrEmptyCharSet     = errors.New("empty character set")
	ErrInvalidDimension = errors.New("invalid dimension for vector generation")
	ErrInvalidRange     = errors.New("invalid range for vector generation")
)

func RandomWord(length int, charSet CharSet) (string, error) {
	if length <= 0 {
		return "",
			fmt.Errorf("%w: word length must be greater than 0, got %d",
				ErrInvalidLength, length)
	}
	if charSet.Len() == 0 {
		return "", ErrEmptyCharSet
	}

	runes := charSet.Runes()
	result := make([]rune, length)
	for i := range result {
		result[i] = runes[rand.IntN(len(runes))]
	}
	return string(result), nil
}

func RandomParagraph(nWords, minWordLen, maxWordLen int, sep string, charSet CharSet) (string, error) {
	if nWords <= 0 {
		return "",
			fmt.Errorf("%w: number of words must be greater than 0, got %d",
				ErrInvalidLength, nWords)
	}

	if charSet.Len() == 0 {
		return "", ErrEmptyCharSet
	}

	if minWordLen < 0 || maxWordLen < 0 {
		return "",
			fmt.Errorf("%w: word length cannot be negative: min %d, max %d",
				ErrInvalidLength, minWordLen, maxWordLen)
	}

	delta := maxWordLen - minWordLen
	if delta < 0 {
		return "",
			fmt.Errorf("%w: minWordLen %d cannot be greater than maxWordLen %d",
				ErrInvalidLength, minWordLen, maxWordLen)
	}

	words := make([]string, nWords)
	for i := range nWords {
		word, err := RandomWord(rand.IntN(delta)+minWordLen, charSet) // Random word length between 1 and 10
		if err != nil {
			return "", err
		}
		words[i] = word
	}
	return strings.Join(words, sep), nil
}

func RandomPassword(length int, charSet CharSet) (string, error) {
	if length <= 0 {
		return "",
			fmt.Errorf("%w: password length must be greater than 0, got %d",
				ErrInvalidLength, length)
	}
	if charSet.Len() == 0 {
		return "", ErrEmptyCharSet
	}

	runes := charSet.Runes()
	result := make([]rune, length)
	for i := range result {
		result[i] = runes[rand.IntN(len(runes))]
	}
	return string(result), nil
}

// RandomUrl generates a random URL with a specified domain and path length.
func RandomUrl(domainLabelCount, pathSegmentCount int, domainCharSet, pathCharSet CharSet) (string, error) {
	if domainLabelCount <= 0 || pathSegmentCount < 0 {
		return "",
			fmt.Errorf("%w: domain length must be greater than 0 and path length cannot be negative, got domain %d, path %d",
				ErrInvalidLength, domainLabelCount, pathSegmentCount)
	}
	if domainCharSet.Len() == 0 {
		return "", ErrEmptyCharSet
	}

	rawDomain := make([]string, domainLabelCount)
	for i := range rawDomain {
		word, err := RandomWord(rand.IntN(10)+3, domainCharSet) // Random word length between 3 and 12
		if err != nil {
			return "", fmt.Errorf("failed to generate random domain word: %w", err)
		}
		rawDomain[i] = word
	}

	rawPath := make([]string, pathSegmentCount)
	for i := range rawPath {
		word, err := RandomWord(rand.IntN(10)+3, pathCharSet) // Random word length between 3 and 12
		if err != nil {
			return "", fmt.Errorf("failed to generate random path word: %w", err)
		}
		rawPath[i] = word
	}

	domain, err := idna.ToASCII(strings.Join(rawDomain, "."))
	if err != nil {
		return "", fmt.Errorf("failed to convert domain to ASCII: %w", err)
	}

	path := url.PathEscape(strings.Join(rawPath, "/"))
	return fmt.Sprintf("%s/%s", domain, path), nil
}

func RandomPGVector(dim int, ub, lb float32) (pgvector.Vector, error) {
	if dim <= 0 {
		return pgvector.Vector{},
			fmt.Errorf("%w: dimension must be greater than 0, got %d",
				ErrInvalidDimension, dim)
	}

	delta := ub - lb
	if delta <= 0 {
		return pgvector.Vector{},
			fmt.Errorf("%w: upper bound %f must be greater than lower bound %f",
				ErrInvalidRange, ub, lb)
	}

	vec := make([]float32, dim)
	for i := range vec {
		vec[i] = rand.Float32()*delta + lb
	}
	return pgvector.NewVector(vec), nil
}

func RandomTime(min, max time.Time) (time.Time, error) {
	if min.After(max) {
		return time.Time{},
			fmt.Errorf("%w: min time %s cannot be after max time %s",
				ErrInvalidRange, min, max)
	}

	delta := max.Sub(min)
	if delta <= 0 {
		return time.Time{},
			fmt.Errorf("%w: max time %s must be after min time %s",
				ErrInvalidRange, max, min)
	}

	randomDuration := time.Duration(rand.Int64N(int64(delta)))
	return min.Add(randomDuration), nil
}
