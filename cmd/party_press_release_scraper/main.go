package main

// import (
// 	"crypto/md5"
// 	"encoding/hex"
// 	"encoding/json"
// 	"fmt"
// 	"os"
// 	"path"
// 	"regexp"
// 	"strings"
// 	"time"

// 	"github.com/ChiaYuChang/weathercock/internal/global"
// 	"github.com/ChiaYuChang/weathercock/internal/scrapers"
// 	flag "github.com/spf13/pflag"
// )

// func ParseKMTPressReleases(output chan<- scrapers.ScrapingResult, extfns map[string]struct{}) error {
// 	// Initialize the scraper with KMT's official site URLs and selectors
// 	return scrapers.ParseKmtOfficialSite(
// 		scrapers.KmtSeedUrls,
// 		scrapers.DefaultBreaks,
// 		scrapers.KmtSelectors,
// 		scrapers.DefaultHeaders,
// 		output,
// 		extfns,
// 	)
// }

// func ParseDPPPressReleases(output chan<- scrapers.ScrapingResult, extfns map[string]struct{}) error {
// 	// Initialize the scraper with DPP's official site URLs and selectors
// 	return scrapers.ParseDppOfficialSite(
// 		scrapers.DppSeedUrls,
// 		scrapers.DefaultBreaks,
// 		scrapers.DppSelectors,
// 		scrapers.DefaultHeaders,
// 		output,
// 		extfns,
// 	)
// }

// func ParseTPPPressReleases(output chan<- scrapers.ScrapingResult, extfns map[string]struct{}) error {
// 	// Initialize the scraper with TPP's official site URLs and selectors
// 	return scrapers.ParseTppOfficialSite(
// 		scrapers.TppSeedUrls,
// 		scrapers.DefaultBreaks,
// 		scrapers.TppSelectors,
// 		scrapers.DefaultHeaders,
// 		output,
// 		extfns,
// 	)
// }

// func main() {
// 	var party string
// 	var dir string
// 	flag.StringVarP(&party, "party", "p", "", "Political party to scrape (kmt, dpp, tpp)")
// 	flag.StringVarP(&dir, "dir", "d", ".", "Directory to save the scraped data (default: current directory)")

// 	flag.Parse()
// 	global.InitBaseLogger()

// 	c := make(chan scrapers.ScrapingResult)
// 	// create a folder for storing the scraped data if it doesn't exist
// 	if _, err := os.Stat(dir); os.IsNotExist(err) {
// 		err = os.MkdirAll(dir, 0755)
// 		if err != nil {
// 			global.Logger.Fatal().
// 				Err(err).
// 				Msgf("Failed to create directory %s for storing scraped data", dir)
// 			os.Exit(1)
// 		}
// 		global.Logger.Info().
// 			Str("directory", dir).
// 			Msg("Created directory for storing scraped data")
// 	}

// 	// read existing extfns in the directory to avoid duplicates
// 	extfns := make(map[string]struct{})
// 	entries, err := os.ReadDir(dir)
// 	if err != nil {
// 		global.Logger.Fatal().
// 			Err(err).
// 			Msgf("Failed to read directory %s for existing files", dir)
// 		os.Exit(1)
// 	}

// 	re := regexp.MustCompile(`(?:\d{4}-\d{2}-\d{2}_|)(\w{32})\.json$`)
// 	for _, entry := range entries {
// 		if !entry.IsDir() {
// 			fn := path.Base(entry.Name())
// 			match := re.FindStringSubmatch(fn)
// 			if len(match) < 2 {
// 				continue
// 			}
// 			global.Logger.Debug().
// 				Str("filename", fn).
// 				Str("hash", match[1]).
// 				Msg("Adding existing file to extfns")
// 			extfns[match[1]] = struct{}{}
// 		}
// 	}

// 	party = strings.ToUpper(party)
// 	go func(party string, exfns map[string]struct{}, c chan scrapers.ScrapingResult) {
// 		var err error
// 		switch party {
// 		case "KMT":
// 			err = ParseKMTPressReleases(c, exfns)
// 		case "DPP":
// 			err = ParseDPPPressReleases(c, exfns)
// 		case "TPP":
// 			err = ParseTPPPressReleases(c, exfns)
// 		default:
// 			global.Logger.Fatal().
// 				Str("party", party).
// 				Msg("Invalid party specified. Use 'kmt', 'dpp', or 'tpp'.")
// 		}
// 		if err != nil {
// 			global.Logger.Fatal().
// 				Err(err).
// 				Msgf("Failed to parse %s press releases", party)
// 			os.Exit(1)
// 		}
// 	}(party, extfns, c)

// 	hasher := md5.New()
// 	logfn := fmt.Sprintf("%s/%s_%s_scraper.log", dir,
// 		time.Now().Format("200601021504"),
// 		strings.ToLower(party))
// 	// create a file to store the log
// 	logf, err := os.Create(logfn)
// 	if err != nil {
// 		global.Logger.Fatal().
// 			Err(err).
// 			Msgf("Failed to create log file %s", logfn)
// 		os.Exit(1)
// 	}
// 	defer logf.Close()

// 	for result := range c {
// 		record := result.ToRecord()
// 		if err := json.NewEncoder(logf).Encode(record); err != nil {
// 			global.Logger.Panic().
// 				Err(err).
// 				Msgf("Failed to write record to log file %s", logfn)
// 			os.Exit(1)
// 		}

// 		if result.Error != nil {
// 			global.Logger.Error().
// 				Err(result.Error).
// 				Msgf("Error scraping %s press release: %s", party, result.Content.Link)
// 			continue
// 		}
// 		if len(result.Warnings) > 0 {
// 			for _, warning := range result.Warnings {
// 				global.Logger.Warn().
// 					Str("link", result.Content.Link).
// 					Msgf("Warning: %s", warning)
// 			}
// 		}
// 		result.Content.Link = strings.TrimPrefix(result.Content.Link, "https://")
// 		hasher.Reset()
// 		hasher.Write([]byte(result.Content.Link))
// 		filename := fmt.Sprintf("%s/%s_%s.json", dir,
// 			result.Content.Date.Format(time.DateOnly),
// 			hex.EncodeToString(hasher.Sum(nil)))
// 		global.Logger.Info().
// 			Str("link", result.Content.Link).
// 			Str("filename", filename).
// 			Msg("[Writer] Successfully scraped press release")

// 		// create file to store the scraped data
// 		go func(fn string, content scrapers.Content) {
// 			file, err := os.Create(fn)
// 			if err != nil {
// 				global.Logger.Error().
// 					Err(err).
// 					Str("filename", fn).
// 					Msg("[Writer] Failed to create file for storing scraped data")
// 			}
// 			defer file.Close()

// 			encoder := json.NewEncoder(file)
// 			encoder.SetIndent("", "  ")
// 			if err := encoder.Encode(content); err != nil {
// 				global.Logger.Error().
// 					Err(err).
// 					Str("filename", fn).
// 					Msg("[Writer] Failed to encode content to JSON")
// 				return
// 			}
// 			global.Logger.Info().
// 				Str("filename", fn).
// 				Msg("[Writer] Successfully saved scraped data to file")
// 		}(filename, result.Content)
// 	}

// 	global.Logger.Info().
// 		Str("party", strings.ToUpper(party)).
// 		Msg("Scraping completed successfully. Press releases have been saved to the database.")
// }
