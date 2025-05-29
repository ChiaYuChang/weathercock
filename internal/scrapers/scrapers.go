package scrapers

import (
	"encoding/json"
	"runtime"
	"time"
)

const (
	UserAgentWinChrome     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0 Safari/537.36"
	UserAgentWinFirefox    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0"
	UserAgentMacChrome     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0 Safari/537.36"
	UserAgentMacFirefox    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7; rv:120.0) Gecko/20100101 Firefox/120.0"
	UserAgentAndroidChrome = "Mozilla/5.0 (Linux; Android 10; Pixel 3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0 Mobile Safari/537.36"
	UserAgentiOSSafari     = "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1"
)

var UserAgents = []string{
	UserAgentWinChrome,
	UserAgentWinFirefox,
	UserAgentMacChrome,
	UserAgentMacFirefox,
	UserAgentAndroidChrome,
	UserAgentiOSSafari,
}

var DefaultUserAgent = UserAgentWinChrome
var DefaultHeaders = map[string]string{
	"User-Agent":      DefaultUserAgent,
	"Accept-Language": "zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7",
	"Accept-Encoding": "gzip",
	"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
	"Connection":      "keep-alive",
	"Referer":         "https://google.com/",
	"Cache-Control":   "no-cache",
}

var DefaultTimeZone, _ = time.LoadLocation("Asia/Taipei")

type Breaks struct {
	ShortBreakMinTime     time.Duration // Minimum time for a short break
	ShortBreakRandomRange time.Duration // Random range added to short break
	LongBreakAfterNPages  int           // Number of pages after which to take a long break
	LongBreakMinTime      time.Duration // Minimum time for a long break
	LongBreakRandomRange  time.Duration // Random range added to long break
}

var DefaultBreaks = Breaks{
	ShortBreakMinTime:     3 * time.Second,
	ShortBreakRandomRange: 2 * time.Second,
	LongBreakAfterNPages:  10,
	LongBreakMinTime:      5 * time.Minute,
	LongBreakRandomRange:  60 * time.Second,
}

// DefaultParallelism is the number of concurrent requests to make.
var DefaultParallelism = runtime.NumCPU() - 1

type Content struct {
	Title    string    `json:"title"`
	Date     time.Time `json:"date"`
	Link     string    `json:"link"`
	Contents []string  `json:"contents"`
}

func (c *Content) MarshalJSON() ([]byte, error) {
	type Alias Content
	return json.Marshal(&struct {
		Date string `json:"date"`
		*Alias
	}{
		Date:  c.Date.Format(time.DateOnly),
		Alias: (*Alias)(c),
	})
}
