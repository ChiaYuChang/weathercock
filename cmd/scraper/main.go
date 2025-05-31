package main

import (
	"os"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/ChiaYuChang/weathercock/internal/scrapers"
)

func main() {
	global.Initialization()
	err := scrapers.ParseKmtOfficialSite(
		scrapers.KmtSeedUrls,
		scrapers.DefaultBreaks,
		scrapers.KmtSelectors,
		scrapers.DefaultHeaders,
	)
	if err != nil {
		global.Logger.Error().
			Err(err).
			Msg("failed to parse KMT official site")
		os.Exit(1)
	}
	global.Logger.Info().
		Msg("KMT official site parsing completed successfully")
}
