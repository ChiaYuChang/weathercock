package scrapers

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/gocolly/colly/v2"
)

const (
	UserAgentWinChrome     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0 Safari/537.36"
	UserAgentWinFirefox    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0"
	UserAgentMacChrome     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0 Safari/537.36"
	UserAgentMacFirefox    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7; rv:120.0) Gecko/20100101 Firefox/120.0"
	UserAgentAndroidChrome = "Mozilla/5.0 (Linux; Android 10; Pixel 3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0 Mobile Safari/537.36"
	UserAgentiOSSafari     = "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1"
)

var (
	ErrPageHasBeenParsed = errors.New("page has already been parsed")
	ErrPageHasNoContent  = errors.New("page has no content")
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

type Delay struct {
	MinDelayTime time.Duration // Minimum time for a short break
	DelayTimeRng time.Duration // Random range added to short break
}

var DefaultBreaks = Delay{
	MinDelayTime: 5 * time.Second,
	DelayTimeRng: 30 * time.Second,
}

// DefaultParallelism is the number of concurrent requests to make.
var DefaultParallelism = runtime.NumCPU() - 1

type SiteSelectors struct {
	TitleSelector            string            `json:"title_selector"`
	ContentContainerSelector string            `json:"content_container_selector"`
	ContentSelector          map[string]string `json:"content_selector"`
	HrefSelector             string            `json:"href_selector"`
	DateTimtSelector         map[string]string `json:"date_time_selector"`
	NextPageTokenSelector    string            `json:"next_page_token_selector,omitempty"`
}

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

type ScrapingResult struct {
	Content  Content  `json:"content"`
	Error    error    `json:"error,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type Record struct {
	Link     string   `json:"link"`
	Status   string   `json:"status"`
	Error    error    `json:"error,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func (s ScrapingResult) ToRecord() Record {
	r := Record{
		Link:     s.Content.Link,
		Status:   "OK",
		Error:    nil,
		Warnings: nil,
	}

	if s.Error != nil {
		if errors.Is(s.Error, ErrPageHasBeenParsed) {
			r.Status = "SKIPPED"
		} else {
			r.Status = "ERROR"
		}
	}
	if len(s.Warnings) > 0 {
		r.Status = "WARNING"
	}

	return r
}

func NewCollector(domain string, maxDepth int, async bool, filter []*regexp.Regexp, breaks Delay,
	headers map[string]string, output chan<- ScrapingResult, files map[string]struct{}) *colly.Collector {
	hasher := md5.New()

	c := colly.NewCollector(
		colly.AllowedDomains(domain),
		colly.URLFilters(filter...),
		colly.Async(async),
		colly.MaxDepth(maxDepth),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  fmt.Sprintf("*%s", domain),
		Parallelism: DefaultParallelism,
		Delay:       breaks.MinDelayTime,
		RandomDelay: breaks.DelayTimeRng,
	})

	c.OnRequest(func(r *colly.Request) {
		hasher.Reset()
		hasher.Write([]byte(strings.TrimLeft(r.URL.String(), "https://")))
		hashsum := hex.EncodeToString(hasher.Sum(nil))
		msg := global.Logger.Debug().
			Str("state", "OnRequest").
			Str("link", strings.TrimLeft(r.URL.String(), "https://")).
			Str("hashsum", hashsum)
		if _, ok := files[hashsum]; ok {
			// Skip the request if the page has already been parsed
			msg.Msg("Skipping parsed page")
			output <- ScrapingResult{
				Content: Content{Link: r.URL.String()},
				Error:   ErrPageHasBeenParsed,
			}
			r.Abort()
			return
		}

		for key, value := range headers {
			r.Headers.Set(key, value)
		}
		msg.Msg("Visiting new page")
	})

	c.OnError(func(r *colly.Response, err error) {
		global.Logger.Error().
			Err(err).
			Str("state", "OnError").
			Int("status_code", r.StatusCode).
			Str("response", string(r.Body)).
			Str("link", r.Request.URL.String()).
			Msg("Request failed")
		output <- ScrapingResult{
			Content: Content{Link: r.Request.URL.String()},
			Error: fmt.Errorf(
				"Request failed with status code %d: %w",
				r.StatusCode, err,
			),
		}
	})

	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode != http.StatusOK {
			global.Logger.Error().
				Str("state", "OnResponse").
				Str("link", r.Request.URL.String()).
				Int("status_code", r.StatusCode).
				Msg("Request failed with non-200 status code")
			return
		}
	})

	return c
}
