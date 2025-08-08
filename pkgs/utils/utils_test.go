package utils_test

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/stretchr/testify/require"
)

func TestJoinSplit(t *testing.T) {
	tcs := []struct {
		name string
		text []string
	}{
		{
			name: "English Text",
			text: []string{
				"This is a test string for joining and splitting.",
				"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
				"Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
				"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
				"Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.",
			},
		},
		{
			name: "Chinese Text",
			text: []string{
				"這是一個用於連接和拆分的測試字符串。",
				"玩想要關係明他們不現在就，買我要很開心大家旁邊滅之刃，的男了什麼都麼都沒好感動，到一來是就是一感覺道位置，多少也不我直接人的但其實生的。",
				"在一起來的藏歡看不好意：身體，認親卡本子一我好，很不就要了很多可愛的精看。嗚嗚嗚，以性還沒長時候底在他人那麼可，一件事狀況所有人我ㄉ，用喜歡以試試，黑說了覺得這件事⋯的名字這幾天錢因為。東西，隨便的比較整理最近決定的事情，後可以啊這覺果真主人，對我來大家的：直接開麼，是源雖然很可以聽賣貨便。",
				"就沒有求直的遺書現在都，比較來克力。啦但自己做到一打完：這件裡面，聽到讓我在可以啊啊。之前是要比較容，廢的笑能要，整問快樂的滿意，的時我今年，也不友想出不是要為到可聖誕，各什麼充場歡的迦爾納才能。",
				"做什麼自己槍為自己⋯靠北一下結果是，人設原因是然能一，都沒悉的看人從頭到，我最近台灣是所以沒⋯取寫得戰的大家。",
			},
		},
	}

	for i, tc := range tcs {
		t.Run(fmt.Sprintf("Case %d %s", i+1, tc.name), func(t *testing.T) {
			result, cuts := utils.Join(tc.text)
			if result == "" {
				t.Error("Expected non-empty result")
			}
			if len(cuts) != len(tc.text) {
				t.Errorf("Expected %d cuts, got %d", len(tc.text), len(cuts))
			}
			require.Equal(t, result, strings.Join(tc.text, ""), "Joined string does not match expected result")

			for i, text := range utils.Split(result, cuts) {
				require.Equal(t, tc.text[i], text, "Split text does not match original text at index %d", i)
			}
		})
	}
}

func TestRandomWord(t *testing.T) {
	tcs := []struct {
		Name    string
		Length  int
		CharSet utils.CharSet
		Err     error
	}{
		{
			Name:    "OK",
			Length:  10,
			CharSet: utils.CharSetLowerCase,
			Err:     nil,
		},
		{
			Name:    "Zero Length",
			Length:  0,
			CharSet: utils.CharSetLowerCase,
			Err:     utils.ErrInvalidLength,
		},
		{
			Name:    "Empty CharSet",
			Length:  10,
			CharSet: utils.CharSet(""),
			Err:     utils.ErrEmptyCharSet,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			word, err := utils.RandomWord(tc.Length, tc.CharSet)
			if tc.Err != nil {
				require.Error(t, err, "Expected error for case: %s", tc.Name)
				return
			}
			require.NoError(t, err, "Unexpected error for case: %s", tc.Name)
			require.Len(t, word, tc.Length, "Generated word length does not match expected length")
			for _, r := range word {
				require.Contains(t, tc.CharSet.Runes(), r, "Generated word contains invalid character")
			}
		})
	}
}

func TestRandomParagraph(t *testing.T) {
	tcs := []struct {
		Name       string
		NWords     int
		MinWordLen int
		MaxWordLen int
		Sep        string
		CharSet    utils.CharSet
		Err        error
	}{
		{
			Name:       "OK",
			NWords:     30,
			MinWordLen: 3,
			MaxWordLen: 10,
			Sep:        " ",
			CharSet:    utils.CharSetLowerCase,
			Err:        nil,
		},
		{
			Name:       "Zero Words",
			NWords:     0,
			MinWordLen: 3,
			MaxWordLen: 10,
			Sep:        " ",
			CharSet:    utils.CharSetLowerCase,
			Err:        utils.ErrInvalidLength,
		},
		{
			Name:       "Negative Min Word Length",
			NWords:     30,
			MinWordLen: -1,
			MaxWordLen: 10,
			Sep:        " ",
			CharSet:    utils.CharSetLowerCase,
			Err:        utils.ErrInvalidLength,
		},
		{
			Name:       "Min Word Length Greater Than Max",
			NWords:     30,
			MinWordLen: 5,
			MaxWordLen: 3,
			Sep:        " ",
			CharSet:    utils.CharSetLowerCase,
			Err:        utils.ErrInvalidLength,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			p, err := utils.RandomParagraph(tc.NWords, tc.MinWordLen, tc.MaxWordLen, tc.Sep, tc.CharSet)
			if tc.Err != nil {
				require.Error(t, err, "Expected error for case: %s", tc.Name)
				return
			}
			require.NoError(t, err, "Unexpected error for case: %s", tc.Name)
			require.NotEmpty(t, p, "Generated paragraph should not be empty")

			for _, word := range strings.Split(p, tc.Sep) {
				require.GreaterOrEqual(t, len(word), tc.MinWordLen, "Generated word is shorter than minimum length")
				require.LessOrEqual(t, len(word), tc.MaxWordLen, "Generated word is longer than maximum length")
				for _, r := range word {
					require.Contains(t, tc.CharSet.Runes(), r, "Generated word contains invalid character")
				}
			}
		})
	}
}

func TestRandomPGVector(t *testing.T) {
	tcs := []struct {
		Name string
		Dim  int
		UB   float32
		LB   float32
		Err  error
	}{
		{
			Name: "OK",
			Dim:  5,
			UB:   1.0,
			LB:   -1.0,
			Err:  nil,
		},
		{
			Name: "Zero Dimension",
			Dim:  0,
			UB:   1.0,
			LB:   -1.0,
			Err:  utils.ErrInvalidDimension,
		},
		{
			Name: "Upper Bound Less Than Lower Bound",
			Dim:  5,
			UB:   -1.0,
			LB:   1.0,
			Err:  utils.ErrInvalidRange,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			vec, err := utils.RandomPGVector(tc.Dim, tc.UB, tc.LB)
			if tc.Err != nil {
				require.Error(t, err, "Expected error for case: %s", tc.Name)
				return
			}
			require.NoError(t, err, "Unexpected error for case: %s", tc.Name)
			require.Len(t, vec.Slice(), tc.Dim, "Generated vector dimension does not match expected dimension")

			iter := slices.All(vec.Slice())
			for _, v := range iter {
				require.GreaterOrEqual(t, v, tc.LB, "Generated vector value is less than lower bound")
				require.LessOrEqual(t, v, tc.UB, "Generated vector value is greater than upper bound")
			}
		})
	}
}

func TestRandomUrl(t *testing.T) {
	tcs := []struct {
		Name          string
		MinParts      int
		MaxParts      int
		DomainCharSet utils.CharSet
		PathCharSet   utils.CharSet
		Err           error
	}{
		{
			Name:     "OK",
			MinParts: 3,
			MaxParts: 5,
			DomainCharSet: utils.MergeCharSets(
				utils.CharSetLowerCase,
				utils.CharSet("使用中文字符集是可以被接受的但是要注意要編碼"),
			),
			PathCharSet: utils.MergeCharSets(
				utils.CharSetAlphaNumeric,
				utils.CharSet("使用中文字符集是可以被接受的但是要注意要編碼"),
			),
			Err: nil,
		},
		{
			Name:          "Zero Parts",
			MinParts:      0,
			MaxParts:      5,
			DomainCharSet: utils.CharSetLowerCase,
			PathCharSet:   utils.CharSetLowerCase,
			Err:           utils.ErrInvalidLength,
		},
		{
			Name:          "Empty CharSet",
			MinParts:      2,
			MaxParts:      5,
			DomainCharSet: utils.CharSet(""),
			PathCharSet:   utils.CharSetLowerCase,
			Err:           utils.ErrEmptyCharSet,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			uStr, err := utils.RandomUrl(tc.MinParts, tc.MaxParts, tc.DomainCharSet, tc.PathCharSet)
			if tc.Err != nil {
				require.Error(t, err, "Expected error for case: %s", tc.Name)
				return
			}
			require.NoError(t, err, "Unexpected error for case: %s", tc.Name)
			require.NotEmpty(t, uStr, "Generated URL should not be empty")

			uStr = "http://" + uStr
			u, err := url.Parse(uStr)
			require.NoError(t, err, "Generated URL is not a valid URL format: %s", uStr)
			require.NotEmpty(t, u.Host, "Generated URL should have a host")
		})
	}
}

func TestPtr(t *testing.T) {
	type People struct {
		Name string
		Age  int
	}

	tsc := []struct {
		name    string
		value   any
		replace any
	}{
		{
			name:    "int",
			value:   42,
			replace: 100,
		},
		{
			name:    "string",
			value:   "Hello, World!",
			replace: "Goodbye, World!",
		},
		{
			name:    "float64",
			value:   3.14,
			replace: 2.718,
		},
		{
			name:    "bool",
			value:   true,
			replace: false,
		},
		{
			name:    "struct",
			value:   People{Name: "Alice", Age: 30},
			replace: People{Name: "Bob", Age: 25},
		},
	}

	for _, tc := range tsc {
		t.Run(tc.name, func(t *testing.T) {
			ptr := utils.Ptr(tc.value)
			require.NotNil(t, ptr, "Pointer should not be nil")
			require.Equal(t, tc.value, *ptr, "Pointer value does not match original value")

			*ptr = tc.replace
			require.Equal(t, tc.replace, *ptr, "Pointer value after replacement does not match expected value")
		})
	}
}
