package subscribers

import (
	"context"
	"fmt"
	"os"

	"github.com/ChiaYuChang/weathercock/internal/workers"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

const (
	LoggerWorkerStreamName  = "TASK"
	LoggerWorkerDurableName = "logger-worker"
	LoggerWorkerSubject     = workers.TaskLog
	LoggerWorkerSource      = "logger-worker"
)

type LoggerWorker struct {
	workers.BaseWorker
	LogFile *os.File
}

func NewLoggerWorker(nc *nats.Conn, logger zerolog.Logger, tracer trace.Tracer, logFilePath string) (*LoggerWorker, error) {
	baseWorker, err := workers.NewBaseWorker(nc, logger, tracer)
	if err != nil {
		return nil, err
	}

	// Ensure the directory for the log file exists
	logDir := "./logs" // TODO: Make configurable
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err = os.MkdirAll(logDir, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
		}
	}

	// Open the log file for appending, create if it doesn't exist
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", logFilePath, err)
	}

	return &LoggerWorker{
		BaseWorker: *baseWorker,
		LogFile:    logFile,
	}, nil
}

func (w *LoggerWorker) Subject() string {
	return LoggerWorkerSubject
}

func (w *LoggerWorker) StreamName() string {
	return LoggerWorkerStreamName
}

func (w *LoggerWorker) DurableName() string {
	return LoggerWorkerDurableName
}

func (w *LoggerWorker) ConsumerOptions() []nats.SubOpt {
	return []nats.SubOpt{
		nats.DeliverNew(),
		nats.AckExplicit(),
		nats.MaxAckPending(1),
		nats.ManualAck(),
	}
}

func (w *LoggerWorker) Handle(ctx context.Context, msg *nats.Msg) error {
	// TODO: Implement logging logic here
	w.Logger.Info().Msg("LoggerWorker received message (not yet implemented)")
	return nil
}

// Close closes the log file when the worker is shut down.
func (w *LoggerWorker) Close() error {
	if w.LogFile != nil {
		return w.LogFile.Close()
	}
	return nil
}
