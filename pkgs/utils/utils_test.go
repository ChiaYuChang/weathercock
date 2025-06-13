package utils_test

import (
	"fmt"
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
