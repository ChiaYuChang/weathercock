// Package kmt provides scraping utilities for the KMT official website.
package scrapers

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/ChiaYuChang/weathercock/pkgs/errors"
	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

const KmtURLTmpl = "https://www.kmt.org.tw/search/label/新聞稿?updated-max=%s&max-results=10#PageNo=%d"

var KmtSeedUrls = []string{
	fmt.Sprintf(KmtURLTmpl, url.QueryEscape(time.Now().Format("2006-01-02T15:04:05+08:00")), 1),
}

func ParseKMTOfficialSite(urls []string, breaks Breaks, headers map[string]string) error {
	const ContentContainerSelector = "body #recentwork #Blog1" // Selector for the main content container
	const MaxDepth = 2

	filters := []*regexp.Regexp{
		regexp.MustCompile(`^https:\/\/www\.kmt\.org\.tw\/search/label/%E6%96%B0%E8%81%9E%E7%A8%BF`),
		regexp.MustCompile(`^https:\/\/www\.kmt\.org\.tw\/\d{4}\/\d{2}\/.*\.html`),
	}

	collector := colly.NewCollector(
		colly.AllowedDomains(
			"www.kmt.org.tw",
		),
		colly.URLFilters(filters...),
		colly.MaxDepth(MaxDepth),
		colly.Async(true),
	)

	collector.Limit(&colly.LimitRule{
		DomainGlob:  "*kmt.org.tw",
		Parallelism: DefaultParallelism,
		Delay:       breaks.ShortBreakMinTime,
		RandomDelay: breaks.ShortBreakRandomRange,
	})

	collector.OnRequest(func(r *colly.Request) {
		for key, value := range headers {
			r.Headers.Set(key, value)
		}
		global.Logger.Info().
			Str("URL", r.URL.String()).
			Msg("Requesting URL")
	})

	collector.OnError(func(r *colly.Response, err error) {
		global.Logger.Error().
			Err(err).
			Int("status_code", r.StatusCode).
			Str("link", r.Request.URL.String()).
			Msg("Request failed")
	})

	collector.OnHTML(
		ContentContainerSelector,
		func(e *colly.HTMLElement) {
			if strings.Contains(e.Request.URL.String(), url.PathEscape("新聞稿")) {
				for _, link := range ParseKMTPressReleaseList(e) {
					collector.Visit(link)
				}
				return
			}
			_, err := ParseKMTPressReleaseContent(e)
			if err != nil {
				global.Logger.Error().
					Err(err).
					Str("link", e.Request.URL.String()).
					Msg("Failed to parse content")
				return
			}
		},
	)

	global.Logger.Info().Msg("Starting scraping process...")
	for _, seed := range urls {
		collector.Visit(seed)
	}
	collector.Wait()
	global.Logger.Info().Msg("Scraping completed, press Ctrl+C to exit")

	return nil
}

func ParseKMTPressReleaseList(e *colly.HTMLElement) (links []string) {
	matches := regexp.MustCompile(`PageNo=(\d+)`).
		FindAllStringSubmatch(e.Request.URL.String(), -1)
	pageNo := 1
	if len(matches) > 0 {
		pageNo, _ = strconv.Atoi(matches[0][1])
	}

	timestamp, ok := e.DOM.Find(".date-posts i.pdt abbr.published[itemprop='datePublished']").
		Last().Attr("title")
	if !ok {
		global.Logger.Error().
			Str("link", e.Request.URL.String()).
			Msg("Failed to find timestamp in document")
		return
	}

	links = make([]string, 0, 10)
	e.DOM.Find(".date-posts h3 a").Each(func(i int, s *goquery.Selection) {
		link, ok := s.Attr("href")
		if !ok {
			return
		}
		links = append(links, link)
	})

	global.Logger.Info().
		Int("page_no", pageNo).
		Int("n_links", len(links)).
		Strs("links", links).
		Str("timestamp", timestamp).
		Msg("Found links")

	e.Request.Visit(fmt.Sprintf(KmtURLTmpl, url.QueryEscape(timestamp), pageNo+1))
	return links
}

func ParseKMTPressReleaseContent(e *colly.HTMLElement) (Content, error) {
	const (
		TitleSelector    = "#div1 h3"                                        // Selector for the article title
		ContentSelector  = "#div1 div.post-body p"                           // Selector for article content paragraphs
		DateTimtSelector = "#div1 div.post-footer-line i.pdt abbr.published" // Selector for the published date
		DateTimeFormat   = time.RFC3339                                      // Expected date format
	)

	content := Content{}
	content.Link = e.Request.URL.String()
	content.Title = utils.NormalizeString(e.DOM.Find(TitleSelector).Text())

	e.DOM.Find(ContentSelector).Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}
		contentText := utils.NormalizeString(s.Text())

		if len(contentText) > 0 {
			content.Contents = append(content.Contents, contentText)
		}
	})

	if len(content.Contents) == 0 {
		global.Logger.Error().
			Str("link", content.Link).
			Msg("No content found")
		err := errors.ErrNoContent.Clone()
		err.Details = append(err.Details, fmt.Sprintf("link: %s", content.Link))
		return content, err
	}

	// Extract date from the page or fallback to content/link
	if dateRaw, ok := e.DOM.Find(DateTimtSelector).Attr("title"); ok {
		content.Date, _ = time.Parse(DateTimeFormat, dateRaw)
	} else {
		if match := regexp.MustCompile(`(\d{2,3})\.(\d{2})\.(\d{2})`).FindStringSubmatch(content.Contents[0]); len(match) == 4 {
			// Try to extract ROC date from content
			year, _ := strconv.Atoi(match[1])
			year += 1911 // convert to ROC year
			month, _ := strconv.Atoi(match[2])
			day, _ := strconv.Atoi(match[3])
			content.Date = time.Date(year, time.Month(month), day, 0, 0, 0, 0, DefaultTimeZone)
		} else {
			// Try to extract date from link, fallback to current time
			re := regexp.MustCompile(`(\d{4})/(\d{2})/blog-post.+\.html`)
			matches := re.FindStringSubmatch(e.Request.URL.String())
			if len(matches) != 3 {
				global.Logger.Warn().
					Str("link", e.Request.URL.String()).
					Msg("failed to extract date from link, using current time")
				content.Date = time.Now()
			} else {
				content.Date, _ = time.ParseInLocation(
					time.DateOnly,
					fmt.Sprintf("%s-%s-01", matches[1], matches[2]),
					DefaultTimeZone)
			}
		}
	}

	s := strings.Join(content.Contents, "\n")
	r := []rune(s)
	if len(r) > 100 {
		s = string(r[:100]) + "..."
	}

	global.Logger.Info().
		Str("link", content.Link).
		Str("title", content.Title).
		Str("date", content.Date.Format("2006-01-02")).
		Str("content", s).
		Msg("Successfully parsed page")
	return content, nil
}
