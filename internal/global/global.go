// Package global provides centralized initialization and configuration for core services.
package global

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Singleton is a generic type that holds a single instance of a type T.
type Singleton[T any] struct {
	instance *T
	once     sync.Once
	errs     []error
}

// NewSingleton creates a new instance of Singleton.
func NewSingleton[T any]() *Singleton[T] {
	return &Singleton[T]{
		instance: new(T),
		once:     sync.Once{},
		errs:     nil,
	}
}

// Errors returns a slice of errors encountered during initialization.
func (s *Singleton[T]) Errors() []error {
	return s.errs
}

func (s *Singleton[T]) Panic(msg string) {
	sb := strings.Builder{}
	for _, err := range templates.errs {
		sb.WriteString(fmt.Sprintf(" - %s\n", err))
	}
	panic(fmt.Errorf("%s:\n%s", msg, sb.String()))
}

func (s *Singleton[T]) CleanUp() {
	s.instance = nil
	s.errs = nil
}

func (s *Singleton[T]) Reset() {
	s.once = sync.Once{}
	s.CleanUp()
}

// Logger is the global zerolog logger instance.
var Logger zerolog.Logger

var templates = NewSingleton[template.Template]()

// Templates returns the singleton instance of parsed go templates.
func Templates() *template.Template {
	templates.once.Do(func() {
		tmpl, err := TemplateRepo(
			TemplateFuncMap(),
			Config().Templates.Path(),
		)
		if err != nil {
			Logger.Error().Err(err).Msg("Failed to parse templates")
			templates.errs = append(templates.errs, fmt.Errorf("failed to parse templates: %w", err))
		}
		Logger.Info().
			Str("dir", Config().Templates.Dir).
			Str("file", Config().Templates.File).
			Msg("Parsed templates successfully")
		templates.instance = tmpl
	})

	if len(templates.errs) > 0 {
		templates.Panic("template parsing errors")
	}
	return templates.instance
}

// mode indicates the current running mode (e.g., "dev", "prod").
var mode string

// SetMode sets the current running mode (e.g., "dev", "prod").
func SetMode(m string) {
	mode = m
}

// Mode returns the current running mode (e.g., "dev", "prod").
func Mode() string {
	return utils.DefaultIfZero(mode, "dev")
}

// natssrv is a singleton for the NATS connection.
var natssrv = NewSingleton[nats.Conn]()

// NATS returns the singleton instance of the NATS connection.
func NATS() *nats.Conn {
	natssrv.once.Do(func() {
		if Config().NATS == nil {
			natssrv.errs = append(natssrv.errs, errors.New("NATS configuration is nil"))
			Logger.Error().Msg("NATS configuration is nil, call LoadConfigs() first")
			return
		}

		server, err := Config().NATS.Connect()
		if err != nil {
			natssrv.errs = append(natssrv.errs,
				fmt.Errorf("failed to connect to NATS server: %w", err))
			Logger.Error().
				Err(natssrv.errs[len(natssrv.errs)-1]).
				Msg("Failed to connect to NATS server")
			return
		}

		if server == nil {
			natssrv.errs = append(natssrv.errs,
				fmt.Errorf("NATS server is nil"))
			Logger.Error().
				Err(natssrv.errs[len(natssrv.errs)-1]).
				Msg("NATS server is nil")
			return
		}
		Logger.Info().
			Str("host", Config().NATS.Host).
			Int("port", Config().NATS.Port).
			Str("username", Config().NATS.Username).
			Str("password", utils.Mask(Config().NATS.Password)).
			Msg("Connected to NATS server")

		for retry := 0; server.Status() != nats.CONNECTED && retry < 5; retry++ {
			wt := 5 * (1 << retry) * time.Second
			Logger.Warn().
				Int("retry", retry).
				Dur("wait_time", wt).
				Msg("Waiting for NATS connection...")
			time.Sleep(wt)
		}

		if server.Status() != nats.CONNECTED {
			natssrv.errs = append(natssrv.errs,
				fmt.Errorf("failed to connect to NATS server after 5 attempts"))
			Logger.Error().
				Err(natssrv.errs[len(natssrv.errs)-1]).
				Msg("Failed to connect to NATS server after 5 attempts")
			return
		}
		Logger.Info().Msg("Successfully pinged NATS server")
		natssrv.instance = server
	})

	if len(natssrv.errs) > 0 {
		natssrv.Panic("NATS connection errors")
	}
	return natssrv.instance
}

// pool is a singleton for the Postgres connection pool.
var pool = NewSingleton[pgxpool.Pool]()

// PostgresPool returns the singleton instance of the Postgres connection pool.
func PostgresPool() *pgxpool.Pool {
	pool.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		p, err := Config().Postgres.Pool(ctx)
		if err != nil {
			pool.errs = append(pool.errs,
				fmt.Errorf("failed to create Postgres connection pool: %w", err))
			Logger.Error().
				Err(pool.errs[len(pool.errs)-1]).
				Msg("Failed to create Postgres connection pool")
			return
		}

		if p == nil {
			pool.errs = append(pool.errs,
				fmt.Errorf("postgres connection pool is nil"))
			return
		}
		Logger.Info().
			Str("host", Config().Postgres.Host).
			Int("port", Config().Postgres.Port).
			Str("database", Config().Postgres.Database).
			Str("username", Config().Postgres.Username).
			Str("password", utils.Mask(Config().Postgres.Password)).
			Bool("sslmode", Config().Postgres.SSLMode).
			Msg("Connected to Postgres DB")

		for retry := 0; p.Ping(ctx) != nil && retry < 5; retry++ {
			wt := 5 * (1 << retry) * time.Second
			Logger.Warn().
				Dur("wait_time", wt).
				Msg("Waiting for Postgres connection...")
			time.Sleep(wt)
		}

		if err := p.Ping(ctx); err != nil {
			pool.errs = append(pool.errs,
				fmt.Errorf("failed to ping to Postgres: %w", err))
		}
		pool.instance = p
	})

	if len(pool.errs) > 0 {
		pool.Panic("Postgres connection pool errors")
	}

	return pool.instance
}

// configuration holds the application configuration.
type configuration struct {
	NATS      *NATSConfig
	Postgres  *PostgresConfig
	Templates *TemplateConfig
}

var config = NewSingleton[configuration]()

// Config returns the singleton instance of the configuration.
// It reads NATS, Postgres, and Template configurations sequentially.
func Config() *configuration {
	config.once.Do(func() {
		c := &configuration{}
		// Initialize NATS configuration
		c.NATS = LoadNATSConfig()
		if err := c.NATS.Validate(); err != nil {
			config.errs = append(config.errs,
				fmt.Errorf("NATS configuration validation failed: %w", err))
			Logger.Error().
				Err(config.errs[len(config.errs)-1]).
				Msg("NATS configuration validation failed")
		} else {
			Logger.Info().Msg("NATS configuration loaded successfully")
		}

		// Initialize Postgres configuration
		c.Postgres = LoadPostgresConfig()
		if err := c.Postgres.ReadPasswordFile(); err != nil {
			config.errs = append(config.errs, err)
			Logger.Error().
				Err(config.errs[len(config.errs)-1]).
				Msg("Failed to read Postgres password file")
		}

		if err := c.Postgres.Validate(); err != nil {
			config.errs = append(config.errs, err)
			Logger.Error().
				Err(config.errs[len(config.errs)-1]).
				Msg("Postgres configuration validation failed")
		} else {
			Logger.Info().Msg("Postgres configuration loaded successfully")
		}

		// Initialize Template configuration
		c.Templates = TemplatesConfig()
		if err := c.Templates.Validate(); err != nil {
			config.errs = append(config.errs, err)
			Logger.Error().
				Err(config.errs[len(config.errs)-1]).
				Msg("Template configuration validation failed")
		} else {
			Logger.Info().Msg("Template configuration loaded successfully")
		}
		config.instance = c
	})

	if len(config.errs) > 0 {
		config.Panic("configuration errors")
	}
	return config.instance
}

// Cache is a global concurrent map for caching arbitrary data.
var Cache = sync.Map{}

// Validate singleton instance
var validate = NewSingleton[validator.Validate]()

// Validate returns the singleton instance of the validator.
func Validator() *validator.Validate {
	validate.once.Do(func() {
		validate.instance = validator.New()
		Logger.Info().Msg("Validator initialized")
		// TODO add self defined validator here
	})

	if len(validate.errs) > 0 {
		validate.Panic("validator errors")
	}
	return validate.instance
}

// ReadDotEnvFile reads a dotfile configuration using Viper.
func ReadDotEnvFile(fname, ftype string, fpath []string) error {
	viper.SetConfigName(fname)
	viper.SetConfigType(ftype)
	for _, p := range fpath {
		viper.AddConfigPath(p)
	}
	return viper.ReadInConfig()
}

// LoadConfigs loads configuration from file and sets up the logger and mode.
func LoadConfigs(fname, ftype string, fpath []string) error {
	if err := ReadDotEnvFile(fname, ftype, fpath); err != nil {
		Logger.Error().Err(err).Msg("Failed to read configuration file")
		return fmt.Errorf("failed to read configuration file: %w", err)
	}
	SetMode(utils.DefaultIfZero(viper.GetString("MODE"), "dev"))
	Logger = InitBaseLogger()
	return nil
}

// InitBaseLogger initializes the base logger for the application.
func InitBaseLogger() zerolog.Logger {
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	logger = logger.Level(utils.IfElse(
		mode == "dev",
		zerolog.DebugLevel,
		zerolog.InfoLevel))

	logger.Info().
		Str("mode", mode).
		Str("log_level", logger.GetLevel().String()).
		Msg("Base Logger Initialized")
	return logger
}

func CleanUp() {
	// Clean up resources
	Cache.Range(func(key, value any) bool {
		Cache.Delete(key)
		return true
	})
	Cache = sync.Map{}

	defer pool.CleanUp()
	if pool.instance != nil {
		pool.instance.Close()
		Logger.Info().Msg("Postgres connection pool closed")
	}

	defer natssrv.CleanUp()
	if natssrv.instance != nil {
		natssrv.instance.Close()
		Logger.Info().Msg("NATS connection closed")
	}

	defer validate.CleanUp()
	defer templates.CleanUp()
	defer config.CleanUp()
}

func Reset() {
	Logger.Warn().Msg("Resetting global state")
	pool.Reset()
	natssrv.Reset()
	validate.Reset()
	templates.Reset()
	config.Reset()
}
