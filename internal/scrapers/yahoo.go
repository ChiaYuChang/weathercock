package scrapers

import (
	"compress/gzip"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/ChiaYuChang/weathercock/pkgs/errors"
	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/PuerkitoBio/goquery"
)

// YahooNewsParseResult holds the result of parsing a Yahoo News article, including timing and error info.
type YahooNewsParseResult struct {
	ParseTime time.Duration    // Time taken to parse the article
	Article   YahooNewsArticle // The parsed article data
	Error     *errors.Error    // Any error encountered during parsing
}

// YahooNewsArticle holds the extracted Yahoo News article data.
type YahooNewsArticle struct {
	ID          string // Unique identifier for the article
	Title       string // Article headline/title
	Url         string // Canonical URL of the article
	Author      string
	Publisher   string
	Description string
	Content     []string
	Keywords    []string
	Published   time.Time
	Modified    time.Time
}

// selectors for extracting article fields from Yahoo News HTML.
const (
	TitleSelector     = "#caas-lead-header-undefined"
	AuthorSelector    = ".caas-attr-meta .caas-attr-item-author span"
	TimeSelector      = ".caas-attr-meta .caas-attr-time-style time"
	PublisherSelector = ".caas-header .caas-logo .caas-attr-provider"
	JSONLDSelector    = ".caas-container script[type='application/ld+json']"
)

// date formats for parsing published and modified times.
const (
	DateTimeFormat       = time.RFC3339
	DateTimeTaiwanFormat = "2006年1月2日 週一 下午3:04"
)

// JSON-LD metadata tags for extracting article fields.
const (
	KeywordsTag      = "keywords"
	DescriptionTag   = "description"
	DatePublishedTag = "datePublished"
	DateModifiedTag  = "dateModified"
)

func Hashing(url string, result *YahooNewsParseResult) string {
	hasher := md5.New()
	hasher.Write([]byte(url))
	hasher.Write([]byte(result.Article.Title))
	hasher.Write([]byte(result.Article.Published.UTC().Format(time.RFC3339)))
	hasher.Write([]byte(result.Article.Modified.UTC().Format(time.RFC3339)))
	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}

func ParseYahooNewsResp(resp *http.Response) *YahooNewsParseResult {
	if resp.StatusCode != http.StatusOK {
		err := errors.NewWithHTTPStatus(
			resp.StatusCode,
			resp.StatusCode,
			fmt.Sprintf("status: %s, failed to fetch Yahoo News article", resp.Status),
			fmt.Sprintf("url: %s", resp.Request.URL.String()),
		)

		return &YahooNewsParseResult{Error: err}
	}

	reader := resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		var err error
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			err := errors.NewWithHTTPStatus(
				http.StatusInternalServerError,
				errors.ECWebpageParsingError,
				"Failed to create gzip reader",
				fmt.Sprintf("err: %s", err.Error()),
				fmt.Sprintf("url: %s", resp.Request.URL.String()),
			)

			return &YahooNewsParseResult{Error: err}
		}
	}
	defer reader.Close()
	result := ParseYahooNewsBody(reader)
	result.Article.ID = Hashing(resp.Request.URL.String(), result)
	return result
}

// ParseYahooNewsBody parses Yahoo News HTML and extracts article fields.
// It attempts to extract metadata from both the HTML and embedded JSON-LD.
// Returns a YahooNewsParseResult with timing and error info.
func ParseYahooNewsBody(r io.Reader) *YahooNewsParseResult {
	tStr := time.Now() // Start timing the parsing process

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		err := errors.Wrap(err,
			http.StatusInternalServerError,
			errors.ECWebpageParsingError,
			"Failed to construct goquery tree from HTML, please ensure the HTML is well-formed and valid",
		)
		return &YahooNewsParseResult{
			ParseTime: time.Since(tStr),
			Error:     err,
		}
	}
	article := &YahooNewsArticle{}

	article.Title = utils.NormalizeString(doc.Find(TitleSelector).Text())
	article.Author = utils.NormalizeString(doc.Find(AuthorSelector).Text())

	// Try to extract JSON-LD metadata if present
	jsonld := doc.Find(JSONLDSelector)
	if jsonld.Length() > 0 {
		var data map[string]interface{}
		text := jsonld.First().Text()
		err = json.Unmarshal([]byte(text), &data)
		if err != nil {
			global.Logger.Warn().
				Err(err).
				Str("jsonld", text).
				Msg("Failed to parse JSON-LD")
		} else {
			if desc, ok := data[DescriptionTag].(string); ok {
				article.Description = utils.NormalizeString(desc)
			}

			// Extract keywords from JSON-LD
			switch kw := data[KeywordsTag].(type) {
			case string:
				article.Keywords = strings.Split(kw, ",")
			case []interface{}:
				for _, keyword := range kw {
					if s, ok := keyword.(string); ok {
						article.Keywords = append(article.Keywords, s)
					}
				}
			default:
				global.Logger.Warn().
					Str(KeywordsTag, fmt.Sprintf("%v", data[KeywordsTag])).
					Msg("Failed to parse keywords")
			}
			sort.Strings(article.Keywords)
			article.Keywords = utils.RemoveDuplicates(article.Keywords)

			// Extract Published and Modified times from JSON-LD
			if timeRaw, ok := data[DatePublishedTag].(string); ok {
				article.Published, err = time.Parse(time.RFC3339, timeRaw)
				if err != nil {
					global.Logger.Warn().
						Err(err).
						Str("time", timeRaw).
						Str("format", time.RFC3339).
						Msg("Failed to parse time from JSON-LD")
				} else {
					global.Logger.Debug().
						Str("time", timeRaw).
						Msg("Parsed time from JSON-LD")
				}
			} else {
				global.Logger.Debug().
					Str("tag", DatePublishedTag).
					Msg("No published time found in JSON-LD")
			}

			// Modified time
			if timeRaw, ok := data[DateModifiedTag].(string); ok {
				article.Modified, err = time.Parse(time.RFC3339, timeRaw)
				if err != nil {
					global.Logger.Warn().
						Err(err).
						Str("time", timeRaw).
						Str("format", time.RFC3339).
						Msg("Failed to parse time from JSON-LD")
				}
			} else {
				global.Logger.Debug().
					Str("tag", DateModifiedTag).
					Msg("No modified time found in JSON-LD")
			}
		}
	}

	// Fallback: try to extract published/modified time from HTML if not found in JSON-LD
	if article.Published.IsZero() {
		timeRaw, ok := doc.Find(TimeSelector).Attr("datetime")
		if !ok {
			timeRaw = doc.Find(TimeSelector).Text()
			// Example: 2025年5月21日 週三 下午4:01
			article.Published, err = time.Parse(DateTimeTaiwanFormat, timeRaw)
			if err != nil {
				global.Logger.Warn().
					Err(err).
					Str("time", timeRaw).
					Str("format", DateTimeTaiwanFormat).
					Msg("Failed to parse time from datetime text")
			}
		} else {
			// Example: 2025-05-21T08:01:50.000Z
			article.Published, err = time.Parse(time.RFC3339, timeRaw)
			if err != nil {
				global.Logger.Warn().
					Err(err).
					Str("time", timeRaw).
					Str("format", DateTimeFormat).
					Msg("Failed to parse time from datetime attribute")
			}
		}

		// Fallback to current time if published time is still zero
		if article.Published.IsZero() {
			global.Logger.Debug().
				Str("time", time.Now().Format(time.DateOnly)).
				Msg("Using current time as published date")
			article.Published = time.Now()
		}
	}

	// Fallback: use published date as modified date if not found
	if article.Modified.IsZero() {
		global.Logger.Debug().
			Msg("Using published date as modified date")
		article.Modified = article.Published
	}

	// Publisher from HTML
	article.Publisher = doc.Find(PublisherSelector).Text()

	// Extract main content paragraphs
	doc.Find(".caas-body p").Each(func(i int, s *goquery.Selection) {
		if s.ChildrenFiltered("span").Length() > 0 {
			span := s.ChildrenFiltered("span").Text()
			// Skip spans that contain certain keywords. These keywords are often used for
			// related articles or additional content that is not part of the main article
			for _, words := range []string{"更多", "延伸閱讀", "相關新聞", "相關報導",
				"相關文章", "相關內容", "延伸內容", "延伸報導", "延伸文章"} {
				if strings.Contains(span, words) {
					return
				}
			}
		}
		text := utils.NormalizeString(s.Text())
		if text != "" {
			article.Content = append(article.Content, text)
		}
	})

	if len(article.Content) == 0 {
		global.Logger.Error().
			Msg("No content found")
		err := errors.ErrNoContent.Clone()
		err.Message = "No content found in Yahoo News article"
		return &YahooNewsParseResult{
			ParseTime: time.Since(tStr),
			Article:   *article,
			Error:     err,
		}
	}

	// Fallback: use first 100 runes of content as description if missing
	if article.Description == "" {
		content := []rune(strings.Join(article.Content, " "))
		if len(content) > 100 {
			content = content[:100]
			content = append(content, []rune("...")...)
		}
		global.Logger.Debug().
			Str("content", string(content)).
			Msg("No description found, using first 100 runes of content")
		article.Description = string(content)
	}

	return &YahooNewsParseResult{
		ParseTime: time.Since(tStr),
		Article:   *article,
		Error:     nil,
	}
}
