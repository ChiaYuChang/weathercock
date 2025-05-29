// Package tpp provides scraping utilities for the KMT official website.
package scrapers

import (
	"compress/gzip"
	"fmt"
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

var TPPSeedUrls = []string{
	"https://www.tpp.org.tw/news?page=%d",
}

func ParseTPPOfficialSite(urls []string, breaks Breaks, headers map[string]string) error {
	// CSS selectors and time format for parsing the TPP website
	const (
		TitleSelector           = "div.content_topic"                  // Selector for the article title
		ContainerSelector       = ".news_container"                    // Selector for the content container
		ContentSelector         = ".content_description > span > span" // Selector for article content paragraphs
		ContentSelectorFallback = ".content_description"               // Fallback selector for content
		HrefSelector            = ".list_frame > a[href]"              // Selector for article links
		DateTimtSelector        = ".content_date"                      // Selector for the published date
		DateTimeFormat          = "2006/01/02"                         // Expected date format
	)

	total, err := TPPRetrieveLastPage("https://www.tpp.org.tw/news", headers)
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

	collector := colly.NewCollector(
		colly.AllowedDomains(
			"www.tpp.org.tw",
		),
		colly.URLFilters(filters...),
		colly.MaxDepth(1),
		colly.Async(true),
	)

	collector.Limit(&colly.LimitRule{
		DomainGlob:  "*tpp.org.tw",
		Parallelism: DefaultParallelism,
		Delay:       breaks.ShortBreakMinTime * time.Second,
		RandomDelay: breaks.ShortBreakRandomRange * time.Second,
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

	collector.OnResponse(func(r *colly.Response) {
		if r.StatusCode != http.StatusOK {
			global.Logger.Error().
				Str("URL", r.Request.URL.String()).
				Int("status_code", r.StatusCode).
				Msg("request failed with non-200 status code")
			return
		}
	})

	collector.OnHTML(
		ContainerSelector,
		func(e *colly.HTMLElement) {
			content := Content{}
			content.Link = e.Request.URL.String()

			date, err := time.ParseInLocation(
				DateTimeFormat,
				e.DOM.Find(DateTimtSelector).First().Text(),
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
			content.Title = utils.NormalizeString(e.DOM.Find(TitleSelector).First().Text())

			if e.DOM.Find(ContentSelector).Length() > 0 {
				e.DOM.Find(ContentSelector).
					Each(func(i int, s *goquery.Selection) {
						text := utils.NormalizeString(s.Text())
						if len(text) > 0 {
							content.Contents = append(content.Contents, text)
						}
					})
			} else {
				global.Logger.Info().
					Str("link", content.Link).
					Msg("no content found, trying fallback selector")

				raw := e.DOM.Find(ContentSelectorFallback).First().Text()
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
		},
	)

	collector.OnHTML(
		HrefSelector,
		func(e *colly.HTMLElement) {
			var link string
			if link = e.Attr("href"); len(link) == 0 {
				return
			}

			for _, filter := range filters {
				if filter.MatchString(link) {
					global.Logger.Info().Msgf("Found link: %s", link)
				}
			}
			collector.Visit(e.Request.AbsoluteURL(link))
		},
	)

	for i := 1; i <= total; i++ {
		err := collector.Visit(fmt.Sprintf(TPPSeedUrls[0], i))
		if err != nil {
			global.Logger.Error().
				Err(err).
				Str("seed_url", fmt.Sprintf(TPPSeedUrls[0], i)).
				Msg("Failed to visit Seed URL")
			return err
		}
	}
	collector.Wait()
	return nil
}

func TPPRetrieveLastPage(u string, headers map[string]string) (int, error) {
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
