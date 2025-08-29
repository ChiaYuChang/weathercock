package main

// import (
// 	"context"
// 	"os/signal"
// 	"syscall"

// 	"github.com/ChiaYuChang/weathercock/internal/global"
// 	"github.com/ChiaYuChang/weathercock/internal/llm"
// 	"github.com/ChiaYuChang/weathercock/internal/storage"
// 	"github.com/ChiaYuChang/weathercock/internal/workers"
// 	"github.com/ChiaYuChang/weathercock/internal/workers/subscribers"
// 	flag "github.com/spf13/pflag"
// )

// func main() {
// 	var configPath string
// 	flag.StringVarP(&configPath, "config", "c", "./configs/workers/keyword_extractor.json", "Path to the configuration file")
// 	flag.Parse()

// 	// Initialize base logger
// 	global.InitBaseLogger()

// 	// Load configurations
// 	cfg, err := global.LoadKeywordExtractorConfig(configPath)
// 	if err != nil {
// 		global.Logger.Fatal().Err(err).Msg("Failed to load keyword extractor config")
// 	}

// 	// Initialize OpenTelemetry
// 	global.InitOTelProvider(cfg.Otel, global.Logger)

// 	// Initialize NATS connection
// 	global.InitNatsConn(cfg.Nats, global.Logger)
// 	defer global.CloseNatsConn()

// 	// Initialize PostgreSQL client
// 	global.InitPostgresClient(cfg.Postgres, global.Logger)
// 	defer global.ClosePostgresClient()

// 	// Initialize Valkey client
// 	global.InitValkeyClient(cfg.Valkey, global.Logger)
// 	defer global.CloseValkeyClient()

// 	// Create storage instance
// 	store := storage.NewStorage(global.PGXPool)

// 	// Create LLM client
// 	llmClient, err := llm.NewClient(cfg.LLM, global.Logger, global.Tracer)
// 	if err != nil {
// 		global.Logger.Fatal().Err(err).Msg("Failed to create LLM client")
// 	}

// 	// Create KeywordExtractorWorker
// 	keywordExtractorWorker, err := subscribers.NewKeywordExtractorWorker(
// 		global.NatsConn,
// 		global.Logger,
// 		global.Tracer,
// 		store,
// 		global.ValkeyClient,
// 		llmClient,
// 	)
// 	if err != nil {
// 		global.Logger.Fatal().Err(err).Msg("Failed to create keyword extractor worker")
// 	}

// 	// Create worker runner
// 	runner, err := workers.NewRunner(
// 		global.NatsConn,
// 		global.Logger,
// 		global.Tracer,
// 		keywordExtractorWorker,
// 		workers.WithTimeout(cfg.Worker.Timeout),
// 		workers.WithHealthCheckPort(cfg.Worker.HealthCheckPort),
// 		workers.WithHealthCheckHost(cfg.Worker.HealthCheckHost),
// 		workers.WithShutdownWaitTime(cfg.Worker.ShutdownWaitTime),
// 	)
// 	if err != nil {
// 		global.Logger.Fatal().Err(err).Msg("Failed to create worker runner")
// 	}

// 	// Set up graceful shutdown
// 	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
// 	defer stop()

// 	global.Logger.Info().Msg("Starting keyword extractor worker...")
// 	if err := runner.Run(ctx); err != nil {
// 		global.Logger.Error().Err(err).Msg("Keyword extractor worker stopped with error")
// 	}

// 	global.Logger.Info().Msg("Keyword extractor worker shut down gracefully.")
// }
