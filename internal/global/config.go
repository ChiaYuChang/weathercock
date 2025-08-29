package global

import (
	"time"
)

type ZeroLogConfig struct {
	GlobalLevel      int8   `json:"global_level"`
	Console          bool   `json:"console"`
	LogFile          string `json:"log_file"`
	IncludeTimestamp bool   `json:"include_timestamp"`
	UseUnixTimestamp bool   `json:"use_unix_timestamp"`
}

type OtelConfig struct {
	ServiceName       string `json:"service_name"`
	CollectorEndpoint string `json:"collector_endpoint"`
	Insecure          bool   `json:"insecure"`
}

type ValkeyConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type WorkerConfig struct {
	Timeout          time.Duration `json:"timeout"`
	HealthCheckPort  int           `json:"health_check_port"`
	HealthCheckHost  string        `json:"health_check_host"`
	ShutdownWaitTime time.Duration `json:"shutdown_wait_time"`
}

type OpenAIConfig struct {
	APIKey  string        `json:"api_key"`
	BaseURL string        `json:"base_url"`
	Model   string        `json:"model"`
	Timeout time.Duration `json:"timeout"`
}

type OllamaConfig struct {
	BaseURL string        `json:"base_url"`
	Model   string        `json:"model"`
	Timeout time.Duration `json:"timeout"`
}

type GeminiConfig struct {
	APIKey  string        `json:"api_key"`
	BaseURL string        `json:"base_url"`
	Model   string        `json:"model"`
	Timeout time.Duration `json:"timeout"`
}

type LLMConfig struct {
	Provider string       `json:"provider"`
	OpenAI   OpenAIConfig `json:"openai"`
	Ollama   OllamaConfig `json:"ollama"`
	Gemini   GeminiConfig `json:"gemini"`
}

type APIConfig struct {
	Name            string         `json:"name"`
	Host            string         `json:"host"`
	Port            int            `json:"port"`
	ShutdownTimeout time.Duration  `json:"shutdown_timeout"`
	Logger          ZeroLogConfig  `json:"logger"`
	Postgres        PostgresConfig `json:"postgres"`
	NATS            NATSConfig     `json:"nats"`
	Valkey          ValkeyConfig   `json:"valkey"`
	Template        TemplateConfig `json:"template"`
	LLM             LLMConfig      `json:"llm"`
	Otel            OtelConfig     `json:"otel"`
}

type MigrateConfig struct {
	Name       string         `json:"name"`
	Postgres   PostgresConfig `json:"postgres"`
	Migrations string         `json:"migrations"`
}

type PartyPressReleaseScraperConfig struct {
	Name     string         `json:"name"`
	Logger   ZeroLogConfig  `json:"logger"`
	Otel     OtelConfig     `json:"otel"`
	Postgres PostgresConfig `json:"postgres"`
}

type ScraperConfig struct {
	Name     string         `json:"name"`
	Logger   ZeroLogConfig  `json:"logger"`
	Postgres PostgresConfig `json:"postgres"`
	Otel     OtelConfig     `json:"otel"`
	NATS     NATSConfig     `json:"nats"`
	Valkey   ValkeyConfig   `json:"valkey"`
	Worker   WorkerConfig   `json:"worker"`
}

type KeywordExtractorConfig struct {
	Name     string         `json:"name"`
	Logger   ZeroLogConfig  `json:"logger"`
	Otel     OtelConfig     `json:"otel"`
	Nats     NATSConfig     `json:"nats"`
	Postgres PostgresConfig `json:"postgres"`
	Valkey   ValkeyConfig   `json:"valkey"`
	Worker   WorkerConfig   `json:"worker"`
	LLM      LLMConfig      `json:"llm"`
}

type LoggerConfig struct {
	Name   string        `json:"name"`
	Logger ZeroLogConfig `json:"logger"`
	Otel   OtelConfig    `json:"otel"`
	Nats   NATSConfig    `json:"nats"`
	Worker WorkerConfig  `json:"worker"`
}
