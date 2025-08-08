package main

import (
	"errors"
	"os"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	var step int
	pflag.IntVarP(&step, "step", "s", 0, "steps looks at the currently active migration version. It will migrate up if n > 0, and down if n < 0.")
	pflag.Parse()

	global.SetMode("dev")
	global.Logger = global.InitBaseLogger()
	global.Logger.Debug().Msg("Logger initialized")

	err := global.ReadDotEnvFile(".env", "env", []string{"."})
	if err != nil {
		global.Logger.
			Err(err).
			Msg("Failed to read .env file")
		os.Exit(1)
	}
	global.Logger.Debug().Msg("Loaded .env file successfully")

	config := global.LoadPostgresConfig()
	global.Logger.Debug().
		Str("conn_str", config.URL()).
		Msg("Postgres config loaded")
	err = config.ReadPasswordFile()
	if err != nil {
		global.Logger.
			Err(err).
			Msg("Failed to read password file")
		os.Exit(1)
	}

	srcURL := "file://" + viper.GetString("MIGRATIONS_PATH")
	dstURL := config.URL()
	global.Logger.Debug().
		Str("src_url", srcURL).
		Str("dst_url", config.URLString()).
		Msg("Migration paths loaded")

	// Initialize migration
	m, err := global.Migrate(srcURL, dstURL)
	if err != nil {
		global.Logger.
			Err(err).
			Msg("Failed to read .env file")
		os.Exit(1)
	}

	if step == 0 {
		global.Logger.Info().
			Msg("No step specified, running migrations up to latest version")
		if err := m.Up(); err != nil {
			if !errors.Is(err, migrate.ErrNoChange) {
				global.Logger.
					Err(err).
					Msg("Failed to apply migrations")
				os.Exit(1)
			}
			global.Logger.Debug().Msg("No new migrations to apply")
		} else {
			global.Logger.Debug().Msg("Migrations applied successfully")
		}
	} else {
		global.Logger.Info().
			Int("step", step).
			Msgf("Running migrations up %d steps", step)
		if err := m.Steps(step); err != nil {
			if !errors.Is(err, migrate.ErrNoChange) {
				global.Logger.
					Err(err).
					Msg("Failed to apply migrations")
				os.Exit(1)
			}
			global.Logger.Debug().Msg("No new migrations to apply")
		} else {
			global.Logger.Debug().Msg("Migrations applied successfully")
		}
	}

	ver, dirty, err := m.Version()
	if err != nil {
		global.Logger.
			Err(err).
			Msg("Failed to get migration version")
		os.Exit(1)
	}
	global.Logger.Debug().
		Uint("version", ver).
		Bool("is_dirty", dirty).
		Msg("Migration version loaded")

}
