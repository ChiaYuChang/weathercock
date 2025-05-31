package scrapers_test

import (
	"testing"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/ChiaYuChang/weathercock/internal/scrapers"
	"github.com/stretchr/testify/require"
)

func TestParseKMTPressRelease(t *testing.T) {
	global.Initialization()
	err := scrapers.ParseKmtOfficialSite(
		scrapers.KmtSeedUrls,
		scrapers.DefaultBreaks,
		scrapers.KmtSelectors,
		scrapers.DefaultHeaders,
	)
	require.NoError(t, err, "Failed to parse KMT press releases")
}

func TestParseTPPPressRelease(t *testing.T) {
	global.Initialization()
	err := scrapers.ParseTppOfficialSite(
		scrapers.TppSeedUrls,
		scrapers.DefaultBreaks,
		scrapers.TppSelectors,
		scrapers.DefaultHeaders,
	)
	require.NoError(t, err, "Failed to parse TPP press releases")
}
