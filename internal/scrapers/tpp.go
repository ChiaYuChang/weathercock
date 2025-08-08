// Package tpp provides scraping utilities for the KMT official website.
package scrapers

import (
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"net/http"
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

// TppSiteSelectors defines the selectors used to extract content from the TPP official site.
var TppSelectors = SiteSelectors{
	TitleSelector:            ".content_topic",
	ContentContainerSelector: ".news_container",
	ContentSelector: map[string]string{
		"default":  ".content_description",
		"fallback": ".content_description span span",
	},
	HrefSelector: ".list_frame > a[href]",
	DateTimtSelector: map[string]string{
		"default": ".content_date",
	},
	NextPageTokenSelector: ".pages_container a:last-child",
}

// TppTimeFormat defines the date format used in TPP press releases.
var TppTimeFormat = "2006/01/02"

// TppSeedUrls contains the initial URLs to start scraping from the TPP official site.
var TppSeedUrls = []string{
	"https://www.tpp.org.tw/news?page=%d",
}

// ParseTppOfficialSite scrapes the TPP official site for press releases.
// Parameters:
// - urls: List of seed URLs to start scraping from. (use TppSeedUrls for default)
// - breaks: Configuration for scraping breaks.
// - selectors: SiteSelectors defining how to extract content from the page. (use TppSelectors for default)
// - headers: HTTP headers to use for requests.
// Returns an error if the scraping process fails.
func ParseTppOfficialSite(urls []string, breaks Delay, selectors SiteSelectors, headers map[string]string, output chan<- ScrapingResult, files map[string]struct{}) error {
	defer close(output)
	total, err := retrieveTppLastPage("https://www.tpp.org.tw/news", headers)
	if err != nil {
		global.Logger.Error().
			Err(err).
			Msg("error while retrieving last page")
		return errors.New(
			http.StatusInternalServerError,
			"failed to retrieve last page",
			err.Error())
	}

	global.Logger.Info().
		Int("total_pages", total).
		Msg("successfully retrieved total pages")
	global.Logger.Info().Msg("Starting scraping process...")

	filters := []*regexp.Regexp{
		regexp.MustCompile(`^https:\/\/www\.tpp\.org\.tw\/newsdetail\/\d{4}$`),
		regexp.MustCompile(`^https:\/\/www\.tpp\.org\.tw\/news.*`),
	}
	collector := NewCollector("www.tpp.org.tw", 2, true, filters,
		breaks, headers, output, files)

	collector.OnHTML(
		selectors.ContentContainerSelector,
		func(e *colly.HTMLElement) {
			content := Content{}
			content.Link = e.Request.URL.String()

			date, err := time.ParseInLocation(
				TppTimeFormat,
				e.DOM.Find(selectors.DateTimtSelector["default"]).First().Text(),
				DefaultTimeZone,
			)
			if err != nil {
				global.Logger.Error().
					Err(err).
					Str("link", content.Link).
					Msg("error parsing date, using current time")
				date = time.Now()
			}
			content.Date = date
			content.Title = utils.NormalizeString(e.DOM.Find(selectors.TitleSelector).First().Text())

			if s, ok := selectors.ContentSelector["default"]; ok && e.DOM.Find(s).Length() > 0 {
				e.DOM.Find(selectors.ContentSelector["default"]).
					Each(func(i int, s *goquery.Selection) {
						text := utils.NormalizeString(s.Text())
						if len(text) > 0 {
							content.Contents = append(content.Contents, text)
						}
					})
			} else {
				global.Logger.Info().
					Str("link", content.Link).
					Msg("no content found with default selector or default selector not set, using fallback selector")
				if s, ok = selectors.ContentSelector["fallback"]; !ok {
					global.Logger.Error().
						Str("link", content.Link).
						Msg("fallback content selector not found, cannot parse content")
					err := errors.ErrNoContent.Clone()
					err.Details = append(err.Details, fmt.Sprintf("link: %s", content.Link))
					output <- ScrapingResult{
						Content: Content{Link: content.Link},
						Error:   err,
					}
					return
				}

				raw := e.DOM.Find(s).First().Text()
				texts := strings.Split(raw, "\n\n")
				for _, text := range texts {
					text = utils.NormalizeString(text)
					if len(text) > 0 {
						content.Contents = append(content.Contents, text)
					}
				}

				if len(content.Contents) == 0 {
					global.Logger.Warn().
						Str("link", content.Link).
						Msg("can not split content into paragraphs, using raw text")
					content.Contents = append(content.Contents, utils.NormalizeString(raw))
				}
			}

			if len(content.Contents) == 0 {
				global.Logger.Warn().
					Str("link", content.Link).
					Msg("no content found")
				output <- ScrapingResult{
					Content: Content{Link: content.Link},
					Warnings: []string{"no content found"},
				}
				return
			}

			c := strings.Join(content.Contents, "\n")
			r := []rune(c)
			global.Logger.Info().
				Str("link", content.Link).
				Str("title", content.Title).
				Str("date", content.Date.Format(time.DateOnly)).
				Str("content", string(r[:min(100, len(r))])).
				Msg("successfully parsed page")
			output <- ScrapingResult{
				Content: content,
			}
		},
	)

	collector.OnHTML(
		selectors.HrefSelector,
		func(e *colly.HTMLElement) {
			var link string
			if link = e.DOM.AttrOr("href", ""); link == "" {
				return
			}

			for _, filter := range filters {
				if filter.MatchString(link) {
					global.Logger.Info().Msgf("Found link: %s", link)
				}
			}

			hasher := md5.New()
			hasher.Write([]byte(link))
			hashsum := hex.EncodeToString(hasher.Sum(nil))
			if _, ok := files[hashsum]; ok {
				global.Logger.Debug().
					Str("link", link).
					Msg("Skipping parsed page")
				output <- ScrapingResult{
					Content: Content{Link: link},
					Error:   ErrPageHasBeenParsed,
				}
				return
			}

			e.Request.Visit(e.Request.AbsoluteURL(link))
			sleep := time.Duration(rand.Int64N(int64(breaks.DelayTimeRng))) + breaks.MinDelayTime
			global.Logger.Debug().
				Int64("duration", int64(sleep/time.Second)).
				Str("link", link).
				Msg("[VisitLoop] Taking a break before visiting next link")
			time.Sleep(sleep)
		},
	)

	for i := 1; i <= total; i++ {
		// if i%10 == 0 {
		// 	delay := breaks.LongBreakMinTime + time.Duration(rand.IntN(int(breaks.LongBreakRandomRange.Seconds())))*time.Second
		// 	global.Logger.Info().
		// 		Int("page", i).
		// 		Dur("delay", delay).
		// 		Msg("Taking a long break before visiting next page")
		// 	time.Sleep(delay)
		// }
		err := collector.Visit(fmt.Sprintf(TppSeedUrls[0], i))
		if err != nil {
			global.Logger.Error().
				Err(err).
				Str("seed_url", fmt.Sprintf(TppSeedUrls[0], i)).
				Msg("Failed to visit Seed URL")
			return err
		}
	}
	collector.Wait()
	return nil
}

// retrieveTppLastPage retrieves the last page number of press releases page from TPP official site.
func retrieveTppLastPage(u string, headers map[string]string) (int, error) {
	// Create HTTP request and set headers
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return 0, errors.New(
			http.StatusInternalServerError,
			"failed to create request",
			err.Error())
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, errors.New(
			http.StatusInternalServerError,
			"failed to send request",
			err.Error())
	}

	reader := resp.Body
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, errors.New(resp.StatusCode, "request failed", resp.Status)
	}

	// Handle gzip encoding if present
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return 0, errors.New(
				http.StatusInternalServerError,
				"failed to create gzip reader",
				err.Error())
		}
	}

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return 0, errors.New(
			http.StatusInternalServerError,
			"html parsing error",
			err.Error())
	}

	lastPageStr := doc.Find(".pages_container a:last-child").AttrOr("href", "")
	if match := regexp.MustCompile(`page=(\d+)`).FindStringSubmatch(lastPageStr); len(match) != 2 {
		return 0, errors.New(
			http.StatusInternalServerError,
			"failed to extract last page number",
			"no last page number found")
	} else {
		lastPageStr = match[1]
	}
	lastPageInt, _ := strconv.Atoi(lastPageStr)
	return lastPageInt, nil
}
