package scrapers_test

import (
	"testing"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/ChiaYuChang/weathercock/internal/scrapers"
	"github.com/stretchr/testify/require"
)

func TestParseKMTPressRelease(t *testing.T) {
	global.InitBaseLogger()

	c := make(chan scrapers.ScrapingResult)
	exfns := make(map[string]struct{})
	err := scrapers.ParseKmtOfficialSite(
		scrapers.KmtSeedUrls,
		scrapers.DefaultBreaks,
		scrapers.KmtSelectors,
		scrapers.DefaultHeaders,
		c,
		exfns,
	)

	for result := range c {
		require.NoError(t, result.Error, "Error in scraping result")
		require.NotEmpty(t, result.Content, "Content should not be empty")
	}
	require.NoError(t, err, "Failed to parse KMT press releases")
}

func TestParseTPPPressRelease(t *testing.T) {
	global.LoadConfigs(".env", "env", []string{"../../"})

	c := make(chan scrapers.ScrapingResult)
	exfns := make(map[string]struct{})
	err := scrapers.ParseTppOfficialSite(
		scrapers.TppSeedUrls,
		scrapers.DefaultBreaks,
		scrapers.TppSelectors,
		scrapers.DefaultHeaders,
		c,
		exfns,
	)

	for result := range c {
		require.NoError(t, result.Error, "Error in scraping result")
		require.NotEmpty(t, result.Content, "Content should not be empty")
	}
	require.NoError(t, err, "Failed to parse TPP press releases")
}
