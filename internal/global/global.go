package global

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"os"
	"sync"
	"time"

	ec "github.com/ChiaYuChang/weathercock/pkgs/errors"
	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// Logger is the global zerolog logger instance.
var Logger zerolog.Logger

// Validator is the global validator instance.
var Validator *validator.Validate

// mode indicates the current running mode (e.g., "dev", "prod").
var mode string

var Cache = sync.Map{}

// SetMode sets the current running mode (e.g., "dev", "prod").
func SetMode(m string) {
	mode = m
}

// Mode returns the current running mode (e.g., "dev", "prod").
func Mode() string {
	return utils.DefaultIfZero(mode, "dev")
}

// InitPostgres initializes the Postgres connection pool and returns it.
func InitPostgres(ctx context.Context, cfg PostgresConfig) (*pgxpool.Pool, error) {
	pCtx, pCancel := context.WithTimeout(ctx, 30*time.Second)
	defer pCancel()

	if err := cfg.Validate(); err != nil {
		return nil, ec.ErrInvalidConfig.Clone().Warp(err).
			WithMessage("invalid postgres config")
	}

	p, err := cfg.Pool(pCtx)
	if err != nil {
		return nil, ec.ErrDBError.Clone().
			WithMessage("failed to create Postgres connection pool").
			Warp(err)
	}

	if p == nil {
		return nil, ec.ErrDBError.Clone().
			WithMessage("failed to create Postgres connection pool").
			WithDetails("postgres connection pool is nil")
	}

	for retry := 0; p.Ping(pCtx) != nil && retry < 5; retry++ {
		wt := 5 * (1 << retry) * time.Second
		Logger.Warn().
			Int("retry", retry).
			Dur("wait_time", wt).
			Msg("Waiting for Postgres connection...")
		time.Sleep(wt)
	}

	if err := p.Ping(pCtx); err != nil {
		return nil, fmt.Errorf("failed to ping to Postgres: %w", err)
	}

	Logger.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Database).
		Str("username", cfg.Username).
		Str("password", utils.Mask(cfg.Password)).
		Bool("sslmode", cfg.SSLMode).
		Msg("connected to Postgres DB")
	return p, nil
}

// InitNATS initializes the NATS connection and returns it.
func InitNATS(cfg NATSConfig) (*nats.Conn, nats.JetStreamContext, error) {
	var conn *nats.Conn
	var js nats.JetStreamContext
	var err error

	if cfg.JetStream {
		conn, js, err = cfg.ConnectJetStream()
	} else {
		conn, err = cfg.Connect()
	}

	if err != nil {
		return nil, nil, ec.ErrNATSConnectionFailed.Clone().
			WithMessage("failed to connect to NATS").
			Warp(err)
	}

	// Common NATS connection checks (ping, etc.)
	for retry := 0; conn.Status() != nats.CONNECTED && retry < 5; retry++ {
		wt := 5 * (1 << retry) * time.Second
		Logger.Warn().
			Int("retry", retry).
			Dur("wait_time", wt).
			Msg("Waiting for NATS connection...")
		time.Sleep(wt)
	}

	if conn.Status() != nats.CONNECTED {
		return nil, nil, ec.ErrNATSServerError.Clone().
			WithMessage("failed to connect to NATS").
			WithDetails("failed to connect to NATS server after 5 attempts")
	}
	Logger.Info().Msg("successfully connected to NATS server")
	return conn, js, nil
}

// InitTemplates initializes and returns the HTML templates.
func InitTemplates(cfg TemplateConfig) (*template.Template, error) {
	tmpl, err := TemplateRepo(
		TemplateFuncMap(),
		cfg.Path(),
	)
	if err != nil {
		return nil, ec.ErrInternalServerError.Clone().
			WithMessage("failed to parse templates").
			Warp(err)
	}
	Logger.Info().
		Str("dir", cfg.Dir).
		Str("file", cfg.File).
		Msg("Parsed templates successfully")
	return tmpl, nil
}

// InitValidator initializes the global validator.
func InitValidator() {
	if Validator == nil {
		Validator = validator.New()
		Logger.Info().Msg("Validator initialized")
		// TODO add self defined validator here
	}
}

// InitOTelProvider initializes the OpenTelemetry TracerProvider and returns the tracer and shutdown function.
func InitOTelProvider(cfg OtelConfig) (trace.Tracer, func(context.Context) error, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tp, err := InitTraceProvider(cfg.CollectorEndpoint, ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize OpenTelemetry trace provider: %w", err)
	}

	tracer := otel.Tracer(cfg.ServiceName)
	Logger.Info().
		Str("service_name", cfg.ServiceName).
		Msg("OpenTelemetry TracerProvider initialized")
	return tracer, tp.Shutdown, nil
}

// InitValkey initializes and returns the Valkey client.
func InitValkey(ctx context.Context, cfg ValkeyConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	pCtx, pCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pCancel()

	_, err := client.Ping(pCtx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Valkey: %w", err)
	}

	Logger.Info().Msg("Connected to Valkey")
	return client, nil
}

// --- Configuration Loading ---

// LoadConfig loads configuration from file into the provided cfg struct.
func LoadConfig(r io.Reader, configType string, cfg any) error {
	if r == nil {
		return ec.ErrInternalServerError.Clone().
			WithMessage("service config reader is nil")
	}

	v := viper.New()
	v.SetDefault("mode", "dev")
	v.SetConfigType(configType)
	if err := v.ReadConfig(r); err != nil {
		return ec.ErrInternalServerError.Clone().
			Warp(err).
			WithMessage("failed to read service config")
	}

	if err := v.Unmarshal(cfg); err != nil {
		return ec.ErrInternalServerError.Clone().
			Warp(err).
			WithMessage("failed to unmarshal service config")
	}

	SetMode(v.GetString("mode"))
	return nil
}

// InitBaseLogger initializes the base logger for the application.
func InitBaseLogger(mode string) zerolog.Logger {
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	logLevel := zerolog.InfoLevel
	if mode == "dev" {
		logLevel = zerolog.DebugLevel
	}
	logger = logger.Level(logLevel)

	logger.Info().
		Str("mode", mode).
		Str("log_level", logger.GetLevel().String()).
		Msg("Base Logger Initialized")
	return logger
}

// InitLogFile opens or creates a log file.
func InitLogFile(fname string) (*os.File, error) {
	f, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open/create log file %s: %w", fname, err)
	}
	return f, nil
}

// NewLogger creates a new logger instance with the given configuration.
func NewLogger(w io.Writer, cfg ZeroLogConfig) (zerolog.Logger, error) {
	if cfg.GlobalLevel < -1 || cfg.GlobalLevel > 7 {
		return zerolog.Nop(), ec.ErrInvalidConfig.Clone().
			WithMessage("invalid global log level, see zerolog docs for help").
			WithDetails(fmt.Sprintf("global log level: %d (should be between -1 and 7)", cfg.GlobalLevel))
	}
	zerolog.SetGlobalLevel(zerolog.Level(cfg.GlobalLevel))

	var writer io.Writer = w
	if cfg.Console {
		writer = zerolog.MultiLevelWriter(w, zerolog.ConsoleWriter{Out: os.Stderr})
	}

	logger := zerolog.New(writer)
	if cfg.IncludeTimestamp {
		logger = logger.With().Timestamp().Logger()
	}

	if cfg.UseUnixTimestamp {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	}

	logger.Info().
		Str("global_level", zerolog.GlobalLevel().String()).
		Bool("console", cfg.Console).
		Bool("include_timestamp", cfg.IncludeTimestamp).
		Msg("Logger service initialized successfully.")

	return logger, nil
}
