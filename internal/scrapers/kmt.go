package scrapers

import (
	"crypto/md5"
	"encoding/hex"
	stde "errors"
	"fmt"
	"math/rand/v2"
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

const KmtURLTmpl = "https://www.kmt.org.tw/search/label/%%E6%%96%%B0%%E8%%81%%9E%%E7%%A8%%BF?updated-max=%s&max-results=10#PageNo=%d"

// SiteSelectors defines the selectors used to extract content from the KMT official
var KmtSelectors = SiteSelectors{
	TitleSelector:            "#div1 h3",
	ContentContainerSelector: "body #recentwork #Blog1",
	ContentSelector: map[string]string{
		"default":  "#div1 div.post-body p",
		"fallback": "#div1 div.post-body description div",
	},
	HrefSelector: ".date-posts h3 a",
	DateTimtSelector: map[string]string{
		"default": "#div1 div.post-footer-line i.pdt abbr.published",
	},
	NextPageTokenSelector: ".date-posts i.pdt abbr.published[itemprop='datePublished']",
}

// KmtTimeFormat defines the date format used in KMT press releases.
var KmtTimeFormat = time.RFC3339

// KmtSeedUrls contains the initial URLs to start scraping from the KMT official site.
var KmtSeedUrls = []string{
	fmt.Sprintf(KmtURLTmpl, url.QueryEscape(time.Now().Format("2006-01-02T15:04:05+08:00")), 1),
}

// ParseKmtOfficialSite scrapes the KMT official site for press releases.
// Parameters:
// - urls: List of seed URLs to start scraping from. (use KmtSeedUrls for default)
// - breaks: Configuration for scraping breaks.
// - selectors: SiteSelectors defining how to extract content from the page. (use KmtSelectors for default)
// - headers: HTTP headers to use for requests.
// Returns an error if the scraping process fails.
func ParseKmtOfficialSite(urls []string, breaks Delay, selectors SiteSelectors,
	headers map[string]string, output chan<- ScrapingResult, files map[string]struct{}) error {
	filters := []*regexp.Regexp{
		regexp.MustCompile(`^https://www\.kmt\.org\.tw/search/label/%E6%96%B0%E8%81%9E%E7%A8%BF`),
		regexp.MustCompile(`^https://www\.kmt\.org\.tw/\d{4}/\d{2}/.*\.html`),
	}
	hasher := md5.New()

	collector := NewCollector("www.kmt.org.tw", 2, true, filters,
		breaks, headers, output, files)

	collector.OnHTML(
		selectors.ContentContainerSelector,
		func(e *colly.HTMLElement) {
			urlStr := e.Request.URL.String()

			if strings.Contains(urlStr, url.PathEscape("新聞稿")) {
				links, next, err := parseKMTPressReleaseList(e, selectors)
				if err != nil {
					global.Logger.Error().
						Err(err).
						Str("link", urlStr).
						Msg("Failed to parse KMT press release list")
					output <- ScrapingResult{
						Content: Content{Link: urlStr},
						Error:   err,
					}
					return
				}
				for _, link := range links {
					if strings.Contains(link, "www.facebook.com") ||
						strings.Contains(link, "www.youtube.com") ||
						strings.Contains(link, "www.instagram.com") ||
						strings.Contains(link, "x.com") {
						global.Logger.Debug().
							Str("src_link", e.Request.URL.String()).
							Str("dst_link", link).
							Msg("Skipping social media link")
						return
					}

					hasher.Reset()
					hasher.Write([]byte(link))
					if _, ok := files[hex.EncodeToString(hasher.Sum(nil))]; ok {
						global.Logger.Debug().
							Str("src_link", e.Request.URL.String()).
							Str("dst_link", link).
							Msg("Skipping already visited link")
						return
					}

					err := e.Request.Visit(link)
					if err != nil {
						// omit errors that are expected
						if stde.Is(err, colly.ErrNoURLFiltersMatch) ||
							stde.Is(err, colly.ErrMaxDepth) ||
							strings.Contains(err.Error(), "already visited") {
							return
						}

						global.Logger.Error().
							Err(err).
							Str("src_link", e.Request.URL.String()).
							Str("dst_link", link).
							Msg("Failed to visit link")
						output <- ScrapingResult{Error: fmt.Errorf(
							"[OnHTML] failed to visit link %s: %w", link, err,
						)}
					}

					sleep := time.Duration(rand.Int64N(int64(breaks.DelayTimeRng))) + breaks.MinDelayTime
					global.Logger.Debug().
						Int64("duration", int64(sleep/time.Second)).
						Str("link", link).
						Msg("[VisitLoop] Taking a break before visiting next link")
					time.Sleep(sleep)
				}
				// new seed
				collector.Visit(next)
			}
			content, err := parseKMTPressReleaseContent(e, selectors)
			if err != nil {
				global.Logger.Error().
					Err(err).
					Str("link", e.Request.URL.String()).
					Msg("Failed to parse content")
				output <- ScrapingResult{
					Error: err,
				}
				return
			}
			output <- ScrapingResult{
				Content: content,
			}
		},
	)

	var err error
	for _, seed := range urls {
		err = collector.Visit(seed)
		if err != nil {
			break
		}
	}
	collector.Wait()
	if err != nil {
		return fmt.Errorf("[Seed] failed to visit seed URLs: %w", err)
	}
	return nil
}

// parseKMTPressReleaseList extracts links and the next page URL from the KMT press release list page.
func parseKMTPressReleaseList(e *colly.HTMLElement, selector SiteSelectors) (links []string, next string, err error) {
	matches := regexp.MustCompile(`PageNo=(\d+)`).FindAllStringSubmatch(e.Request.URL.String(), -1)
	pageNo := 1
	if len(matches) > 0 {
		pageNo, _ = strconv.Atoi(matches[0][1])
	}

	timestamp, ok := e.DOM.Find(selector.NextPageTokenSelector).Last().Attr("title")
	if !ok {
		global.Logger.Error().
			Str("link", e.Request.URL.String()).
			Msg("Failed to find timestamp in document")
		return nil, "", fmt.Errorf("failed to find timestamp in document for link: %s", e.Request.URL.String())
	}

	links = []string{}
	e.DOM.Find(selector.HrefSelector).Each(func(i int, s *goquery.Selection) {
		link, ok := s.Attr("href")
		if ok && link != "" {
			links = append(links, link)
		}
	})

	global.Logger.Info().
		Int("page_no", pageNo).
		Int("n_links", len(links)).
		Strs("links", links).
		Str("timestamp", timestamp).
		Msg("Found links")

	next = fmt.Sprintf(KmtURLTmpl, url.QueryEscape(timestamp), pageNo+1)
	return links, next, nil
}

// parseKMTPressReleaseContent extracts the title, date, and content from a KMT press release page.
func parseKMTPressReleaseContent(e *colly.HTMLElement, selector SiteSelectors) (Content, error) {
	content := Content{}
	content.Link = e.Request.URL.String()
	content.Title = utils.NormalizeString(e.DOM.Find(selector.TitleSelector).Text())

	e.DOM.Find(selector.ContentSelector["default"]).
		Each(func(i int, s *goquery.Selection) {
			if i == 0 {
				return
			}
			contentText := utils.NormalizeString(s.Text())
			if len(contentText) > 0 {
				content.Contents = append(content.Contents, contentText)
			}
		})

	if len(content.Contents) == 0 {
		global.Logger.Warn().
			Str("link", content.Link).
			Str("selector", selector.ContentSelector["default"]).
			Msg("No content found by default selector, try fallback selector")
		if s, ok := selector.ContentSelector["fallback"]; ok && e.DOM.Find(s).Length() > 0 {
			e.DOM.Find(s).Each(func(i int, s *goquery.Selection) {
				contentText := utils.NormalizeString(s.Text())
				if len(contentText) > 0 {
					content.Contents = append(content.Contents, contentText)
				}
			})
		} else {
			global.Logger.Error().
				Str("link", content.Link).
				Bool("has_fallback_selector", ok).
				Str("selector", selector.ContentSelector["fallback"]).
				Msg("No content found by fallback selector, cannot parse content")
			err := errors.ErrNoContent.Clone()
			err.Details = append(err.Details, fmt.Sprintf("link: %s", content.Link))
			return content, err
		}
	}

	// Extract date from the page or fallback to content/link
	if dateRaw, ok := e.DOM.Find(selector.DateTimtSelector["default"]).Attr("title"); ok {
		content.Date, _ = time.Parse(KmtTimeFormat, dateRaw)
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
