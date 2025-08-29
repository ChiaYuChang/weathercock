package cfggen

import (
	"fmt"
	"io"

	"github.com/spf13/viper"
)

type CfgGen struct {
	dst *viper.Viper // Destination viper instance for building the output config
	src *viper.Viper // Source viper instance for reading environment variables (e.g., .env)
}

// NewCfgGen creates a new CfgGen instance, taking a source viper instance
// that has already loaded the environment variables (e.g., from .env).
func NewCfgGen(src *viper.Viper) *CfgGen {
	return &CfgGen{
		dst: viper.New(), // Create a new viper instance for the generated config
		src: src,
	}
}

func (c *CfgGen) WriteTo(w io.Writer, t string) error {
	c.dst.SetConfigType(t)
	if err := c.dst.WriteConfigTo(w); err != nil {
		return fmt.Errorf("error writing config: %w", err)
	}
	return nil
}

func (c *CfgGen) AddZerologLoggerConfig() {

}

func (c *CfgGen) AddPostgresConfig() {
	c.dst.SetDefault("postgres.username", "postgres")
	c.dst.SetDefault("postgres.host", "localhost")
	c.dst.SetDefault("postgres.port", 5432)
	c.dst.SetDefault("postgres.sslmode", false)

	c.dst.Set("postgres.username", c.src.GetString("POSTGRES_USER"))
	c.dst.Set("postgres.password_file", c.src.GetString("POSTGRES_PASSWORD_FILE"))
	c.dst.Set("postgres.host", c.src.GetString("POSTGRES_HOST"))
	c.dst.Set("postgres.port", c.src.GetInt("POSTGRES_PORT"))
	c.dst.Set("postgres.sslmode", c.src.GetBool("POSTGRES_SSL_MODE"))
	c.dst.Set("postgres.dbname", c.src.GetString("POSTGRES_APP_DB"))
}

func (c *CfgGen) AddMigrationConfig() {
	c.dst.Set("migrations.path", c.src.GetString("MIGRATIONS_PATH"))
	c.dst.Set("migrations.dbname", c.src.GetString("MIGRATIONS_DB"))
}

func (c *CfgGen) AddTemplatesConfig() {
	c.dst.SetDefault("templates.path", "templates")
	c.dst.SetDefault("templates.file", "*.gotmpl")

	c.dst.Set("templates.path", c.src.GetString("TMPL_DIR"))
	c.dst.Set("templates.file", c.src.GetString("TMPL_FILE"))
}

func (c *CfgGen) AddValkeyConfig() {
	c.dst.SetDefault("valkey.host", "localhost")
	c.dst.SetDefault("valkey.port", 6379)

	c.dst.Set("valkey.host", c.src.GetString("VALKEY_HOST"))
	c.dst.Set("valkey.port", c.src.GetInt("VALKEY_PORT"))
}

func (c *CfgGen) AddNATSConfig() {
	c.dst.SetDefault("nats.host", "localhost")
	c.dst.SetDefault("nats.port", 8222)

	c.dst.Set("nats.host", c.src.GetString("NATS_HOST"))
	c.dst.Set("nats.port", c.src.GetInt("NATS_HTTP_PORT")) // Please confirm if this should be NATS_PORT instead of NATS_HTTP_PORT
	c.dst.Set("nats.username", c.src.GetString("NATS_USER"))
	c.dst.Set("nats.password", c.src.GetString("NATS_PASS"))
}

func (c *CfgGen) AddLLMConfig() {
	c.dst.SetDefault("ollama.host", "localhost")
	c.dst.SetDefault("ollama.port", 11434)
	c.dst.SetDefault("openai.base_url", "https://api.openai.com/v1")

	c.dst.Set("gemini.api_key", c.src.GetString("GEMINI_API_KEY"))
	c.dst.Set("ollama.host", c.src.GetString("OLLAMA_HOST"))
	c.dst.Set("ollama.port", c.src.GetInt("OLLAMA_PORT"))
	c.dst.Set("openai.api_key", c.src.GetString("OPENAI_API_KEY"))
	c.dst.Set("openai.base_url", c.src.GetString("OPENAI_BASE_URL"))
}

func (c *CfgGen) AddScraperConfig() {
	c.dst.Set("workers.scraper.otel_service_name", c.src.GetString("SCRAPER_WORKER_OTEL_SERVICE_NAME"))
	c.dst.Set("workers.scraper.otel_collector_ep", c.src.GetString("SCRAPER_WORKER_OTEL_COLLECTOR_EP"))
	c.dst.Set("workers.scraper.otel_insecure", c.src.GetBool("SCRAPER_WORKER_OTEL_INSECURE"))
}

func (c *CfgGen) AddLoggerConfig() {
	c.dst.Set("workers.logger.otel_service_name", c.src.GetString("LOGGER_WORKER_OTEL_SERVICE_NAME"))
	c.dst.Set("workers.logger.otel_collector_ep", c.src.GetString("LOGGER_WORKER_OTEL_COLLECTOR_EP"))
	c.dst.Set("workers.logger.otel_insecure", c.src.GetBool("LOGGER_WORKER_OTEL_INSECURE"))
}

func (c *CfgGen) AddKeywordExtractorConfig() {
	c.dst.Set("workers.keywords_extractor.otel_service_name", c.src.GetString("KEYWORD_EXTRACTOR_WORKER_OTEL_SERVICE_NAME"))
	c.dst.Set("workers.keywords_extractor.otel_collector_ep", c.src.GetString("KEYWORD_EXTRACTOR_WORKER_OTEL_COLLECTOR_EP"))
	c.dst.Set("workers.keywords_extractor.otel_insecure", c.src.GetBool("KEYWORD_EXTRACTOR_WORKER_OTEL_INSECURE"))
}
