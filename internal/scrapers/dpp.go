package scrapers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

const DppURLTmpl = "https://www.dpp.org.tw/%s"

var DppSeedUrls = []string{
	fmt.Sprintf(DppURLTmpl, "media"),
	fmt.Sprintf(DppURLTmpl, "anti_rumor"),
}

func ParseDPPOfficialSite(urls []string, breaks Breaks, headers map[string]string) error {
	// CSS selectors and time format for parsing the DPP website
	const (
		TitleSelector        = "h2"
		ContainerSelector    = "article.news_content"
		MediaContentSelector = "#media_contents"
		AntiRumorSelector    = "#news_contents"
		DateTimeFormat       = "2006-01-02"
		DateTimeSelector     = "p.news_content_date"
		HrefSelector         = "a[href]"
	)

	filters := []*regexp.Regexp{
		regexp.MustCompile(`^https://www\.dpp\.org\.tw/(?:media|anti_rumor)`),
	}

	collector := colly.NewCollector(
		colly.AllowedDomains(
			"www.dpp.org.tw",
		),
		colly.Async(true),
		colly.URLFilters(filters...),
		colly.MaxDepth(1),
		colly.Async(true),
	)

	collector.Limit(&colly.LimitRule{
		DomainGlob:  "*dpp.org.tw",
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
				e.DOM.Find(DateTimeSelector).First().Text(),
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

			var contentContainerID string
			if strings.Contains(content.Link, "media") {
				contentContainerID = MediaContentSelector
			}

			if strings.Contains(content.Link, "anti_rumor") {
				contentContainerID = AntiRumorSelector
			}

			e.DOM.Find(contentContainerID).Children().
				Filter("p").Each(func(i int, s *goquery.Selection) {
				text := utils.NormalizeString(s.Text())
				if len(text) > 0 {
					content.Contents = append(content.Contents, text)
				}
			})

			if len(content.Contents) == 0 {
				global.Logger.Error().
					Str("link", content.Link).
					Msg("No content found")
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
			if link = e.Attr("href"); link == "" {
				return
			}

			for _, filter := range filters {
				if filter.MatchString(link) {
					global.Logger.Info().Msgf("Found link: %s", link)
				}
			}

			err := e.Request.Visit(e.Request.AbsoluteURL(link))
			if err != nil {
				global.Logger.Error().
					Err(err).
					Str("src_link", e.Request.URL.String()).
					Str("dst_link", link).
					Msg("Failed to visit link")
			}
		},
	)

	for _, url := range urls {
		err := collector.Visit(url)
		if err != nil {
			global.Logger.Error().
				Err(err).
				Str("seed_url", url).
				Msg("Failed to visit Seed URL")
			return err
		}
	}
	collector.Wait()
	return nil
}
