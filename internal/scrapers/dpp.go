package scrapers

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

const DppURLTmpl = "https://www.dpp.org.tw/%s"

var DppSelectors = SiteSelectors{
	TitleSelector:            "h2",
	ContentContainerSelector: "article.news_content",
	ContentSelector: map[string]string{
		"media":      "#media_contents > p",
		"anti_rumor": "#news_contents > p",
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

func ParseDppOfficialSite(urls []string, breaks Delay, selectors SiteSelectors, headers map[string]string) error {
	collector := newCollector(
		"www.dpp.org.tw", 2, true,
		[]*regexp.Regexp{
			regexp.MustCompile(`^https://www\.dpp\.org\.tw/(?:media|anti_rumor)`),
		}, breaks, headers)

	collector.OnHTML(
		selectors.ContentContainerSelector,
		func(e *colly.HTMLElement) {
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
					Str("link", content.Link).
					Msg("error parsing date, using current time")
				date = time.Now()
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
		selectors.HrefSelector,
		func(e *colly.HTMLElement) {
			var link string
			if link = e.Attr("href"); link == "" {
				return
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
