package scrapers

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
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

const DppURLTmpl = "https://www.dpp.org.tw/%s"

var DppSelectors = SiteSelectors{
	TitleSelector:            "h2",
	ContentContainerSelector: "article.news_content",
	ContentSelector: map[string]string{
		"media":      "#media_contents",
		"anti_rumor": "#news_contents",
	},
	HrefSelector: "a[href]",
	DateTimtSelector: map[string]string{
		"default": "p.news_content_date",
	},
}

var DppTimeFormat = "2006-01-02"

var DppSeedUrls = []string{
	fmt.Sprintf(DppURLTmpl, "media"),
	fmt.Sprintf(DppURLTmpl, "anti_rumor"),
}

func ParseDppOfficialSite(urls []string, breaks Delay, selectors SiteSelectors,
	headers map[string]string, output chan<- ScrapingResult, files map[string]struct{},
) error {
	// Ensure the output channel is closed when done
	defer close(output)
	hasher := md5.New()

	subjects := []struct {
		name     string
		subject  string
		selector string
		latest   int
		oldest   int
	}{
		{"press meida", "media", ".news_abtn", 0, 8},
		{"rumor", "anti_rumor", ".event828_news_item > a", 0, 3},
	}

	re := regexp.MustCompile(`www\.dpp\.org\.tw/(?:anti_rumor|media)/contents/(\d+)`)
	for i, subject := range subjects {
		resp, err := http.Get(fmt.Sprintf(DppURLTmpl, subject.subject))
		if err != nil {
			return fmt.Errorf("faile to fetch the latest %s", subject.name)
		}

		if resp.StatusCode != http.StatusOK {
			var content string
			if resp.Body != nil {
				defer resp.Body.Close()
				if body, err := io.ReadAll(resp.Body); err == nil {
					content = string(body)
				}
			}

			return fmt.Errorf(
				"failed to fetch page: %s, status code: %d (content: %q)",
				resp.Request.URL, resp.StatusCode, content)
		}
		defer resp.Body.Close()
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to parse page %s: %w", resp.Request.URL, err)
		}

		elem := doc.Find(subject.selector).First()
		if href, ok := elem.Attr("href"); ok {
			match := re.FindStringSubmatch(href)
			id, err := strconv.Atoi(match[1])
			if err != nil {
				return fmt.Errorf("failed to parse %s ID from link: %s", subject.name, href)
			}

			global.Logger.Info().
				Str("subject", subject.name).
				Str("link", href).
				Int("id", id).
				Msg("Found latest news link")
			subjects[i].latest = id
		} else {
			global.Logger.Error().
				Str("subject", subject.name).
				Msg("Failed to find latest news link")
			return fmt.Errorf("failed to find latest %s link", subject.name)
		}
	}

	collector := NewCollector(
		"www.dpp.org.tw", 2, true,
		[]*regexp.Regexp{
			regexp.MustCompile(`^https://www\.dpp\.org\.tw/(?:media|anti_rumor)`),
		}, breaks, headers, output, files)

	collector.OnHTML(
		selectors.ContentContainerSelector,
		func(e *colly.HTMLElement) {
			result := ScrapingResult{}

			content := Content{}
			content.Link = e.Request.URL.String()

			date, err := time.ParseInLocation(
				DppTimeFormat,
				e.DOM.Find(selectors.DateTimtSelector["default"]).First().Text(),
				DefaultTimeZone,
			)
			if err != nil {
				global.Logger.Error().
					Err(err).
					Str("state", "OnHTML").
					Str("link", content.Link).
					Msg("error parsing date, using current time")
				date = time.Now()
				result.Warnings = append(
					result.Warnings,
					fmt.Sprintf("error parsing date for link %s: %v", content.Link, err),
					fmt.Sprintf("using current time: %s", date.Format(time.RFC3339)),
				)
			}
			content.Date = date
			content.Title = utils.NormalizeString(
				e.DOM.Find(selectors.TitleSelector).First().Text())

			var contentContainerID string
			for _, cat := range []string{"media", "anti_rumor"} {
				if strings.Contains(content.Link, cat) {
					contentContainerID = selectors.ContentSelector[cat]
					break
				}
			}

			if contentContainerID == "" {
				global.Logger.Error().
					Str("state", "OnHTML").
					Str("link", content.Link).
					Msg("No content container ID found for DPP press release")
				err := errors.ErrNoContent.Clone()
				err.Details = append(err.Details, fmt.Sprintf("link: %s", content.Link))
				output <- ScrapingResult{
					Content: Content{Link: content.Link},
					Error:   err,
				}
				return
			}

			elems := e.DOM.Find(contentContainerID).Children().Filter("p")
			if elems.Length() == 0 || len(utils.NormalizeString(elems.Text())) == 0 {
				global.Logger.Warn().
					Str("state", "OnHTML").
					Str("link", content.Link).
					Msg("No <p> elements found, trying to find <div> elements instead")
				// If no <p> elements found, try to find <div> elements
				elems = e.DOM.Find(contentContainerID).Children().Filter("div")
			}
			elems.Each(func(i int, s *goquery.Selection) {
				text := utils.NormalizeString(s.Text())
				if len(text) > 0 {
					content.Contents = append(content.Contents, text)
				}
			})

			if len(content.Contents) == 0 {
				global.Logger.Error().
					Str("link", content.Link).
					Str("title", content.Title).
					Msg("No content found")
				err := errors.ErrNoContent.Clone()
				err.Details = append(err.Details, fmt.Sprintf("link: %s", content.Link))
				output <- ScrapingResult{
					Content: Content{Link: content.Link},
					Error:   err,
				}
				return
			}
			output <- ScrapingResult{
				Content: content,
			}
		},
	)

	var err error
	for _, subject := range subjects {
		for i := subject.latest; i >= subject.oldest; i-- {
			link := fmt.Sprintf(DppURLTmpl+"/contents/%d", subject.subject, i)

			linkWithoutScheme := strings.TrimLeft(link, "https://")
			hasher.Reset()
			hasher.Write([]byte(linkWithoutScheme))
			hashsum := hex.EncodeToString(hasher.Sum(nil))
			global.Logger.Debug().
				Str("link", linkWithoutScheme).
				Str("hashsum", hashsum).
				Msg("Checking if link has been parsed")
			if _, ok := files[hashsum]; ok {
				global.Logger.Debug().
					Str("link", link).
					Msg("Skipping parsed page")
				output <- ScrapingResult{
					Content: Content{Link: link},
					Error:   ErrPageHasBeenParsed,
				}
				continue
			}
			err = collector.Visit(link)
			if err != nil {
				err = fmt.Errorf("[Seed] Failed to visit DPP URL %s: %w", link, err)
				break
			}
			sleep := time.Duration(rand.Int64N(int64(breaks.DelayTimeRng))) + breaks.MinDelayTime
			global.Logger.Debug().
				Int64("duration", int64(sleep/time.Second)).
				Str("link", link).
				Msg("[OnHTML] Taking a break before visiting next link")
			time.Sleep(sleep)
		}
	}
	collector.Wait()
	if err != nil {
		return err
	}
	return nil
}
