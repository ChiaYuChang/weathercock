package global

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"

	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

// PostgresConfig holds the configuration for connecting to a PostgreSQL database.
// All fields are required except for PasswordFile and SSLMode.
// PasswordFile is used to specify a file from which the password can be read.
// The Password field is required unless PasswordFile is provided.
// SSLMode is a boolean that indicates whether to use SSL for the connection. (default: false)
type PostgresConfig struct {
	Host         string `json:"host"          validate:"required"                      mapstructure:"host"`
	Port         int    `json:"port"          validate:"required"                      mapstructure:"port"`
	Username     string `json:"username"      validate:"required"                      mapstructure:"username"`
	Password     string `json:"password"      validate:"required_without=PasswordFile" mapstructure:"password"`
	PasswordFile string `json:"password_file" validate:"required_without=Password"     mapstructure:"password_file"`
	Database     string `json:"database"      validate:"required"                      mapstructure:"database"`
	SSLMode      bool   `json:"sslmode"                                                mapstructure:"sslmode"`
}

func LoadPostgresConfig() *PostgresConfig {
	viper.SetDefault("POSTGRES_HOST", "localhost")
	viper.SetDefault("POSTGRES_PORT", 5432)
	viper.SetDefault("POSTGRES_USER", "postgres")
	viper.SetDefault("POSTGRES_APP_DB", "db")
	viper.SetDefault("POSTGRES_SSLMODE", false)

	cfx := &PostgresConfig{
		Host:         viper.GetString("POSTGRES_HOST"),
		Port:         viper.GetInt("POSTGRES_PORT"),
		Username:     viper.GetString("POSTGRES_USER"),
		Password:     viper.GetString("POSTGRES_PASSWORD"),
		PasswordFile: viper.GetString("POSTGRES_PASSWORD_FILE"),
		Database:     viper.GetString("POSTGRES_APP_DB"),
		SSLMode:      viper.GetBool("POSTGRES_SSLMODE"),
	}

	if err := cfx.ReadPasswordFile(); err != nil {
		Logger.Warn().Err(err).Msg("failed to read password file")
		return nil
	}
	return cfx
}

// MarshalJSON is a custom JSON marshaller that masks the password field.
func (c PostgresConfig) MarshalJSON() ([]byte, error) {
	password := c.Password
	if c.PasswordFile != "" {
		password = utils.Mask(password)
	}

	type Alias PostgresConfig
	return json.Marshal(&struct {
		Password string `json:"password,omitempty"`
		*Alias
	}{
		Password: password,
		Alias:    (*Alias)(&c),
	})
}

// MarshalJSONPlain is a custom JSON marshaller that does not mask the password.
// It is used for internal purposes where the password needs to be visible.
// Use with caution, as it will expose the password in the JSON output.
// This should not be used in production or for public APIs.
func (c PostgresConfig) MarshalJSONPlain() ([]byte, error) {
	type Alias PostgresConfig
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(&c),
	})
}

// ReadPasswordFile reads the password from a file specified in the PasswordFile field.
// If PasswordFile is empty, it does nothing.
func (c *PostgresConfig) ReadPasswordFile() error {
	if c.PasswordFile == "" {
		return nil
	}

	data, err := os.ReadFile(c.PasswordFile)
	if err != nil {
		return fmt.Errorf("failed to read password file %s: %w", c.PasswordFile, err)
	}

	password := strings.TrimSpace(string(data))
	if c.Password != "" {
		Logger.Warn().
			Str("password", utils.Mask(c.Password)).
			Str("password_from_file", utils.Mask(password)).
			Msg("password provided in config will be replaced by password from file")
	}
	c.Password = password
	return nil
}

// URL returns the PostgreSQL connection string based on the configuration.
func (c *PostgresConfig) URL() string {
	sslmode := "disable"
	if c.SSLMode {
		sslmode = "enable"
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Username, c.Password, c.Host, c.Port, c.Database, sslmode)
}

// URLString returns the PostgreSQL connection string based on the configuration.
// It masks the password in the connection string.
func (c *PostgresConfig) URLString() string {
	sslmode := "disable"
	if c.SSLMode {
		sslmode = "enable"
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Username, strings.Repeat("●", rand.IntN(10)+5), // Mask password in URL
		c.Host, c.Port, c.Database, sslmode)
}

// Pool returns a connection pool for the PostgreSQL database.
func (c *PostgresConfig) Pool(ctx context.Context) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, c.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Postgres: %w", err)
	}
	return pool, nil
}

// String returns a JSON representation of the PostgresConfig.
// It masks the password if PasswordFile is set, otherwise it shows the actual password.
func (c PostgresConfig) String() string {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Sprintf("PostgresConfig{Host: %s, Port: %d, Username: %s, Database: %s, SSLMode: %t}",
			c.Host, c.Port, c.Username, c.Database, c.SSLMode)
	}
	return string(b)
}

// Validate checks the PostgresConfig for required fields and conditions.
// It is recommended to call ReadPasswordFile() before calling this method.
func (c *PostgresConfig) Validate() error {
	if err := Validator.Struct(c); err != nil {
		return fmt.Errorf("invalid Postgres configuration: %w", err)
	}

	if len(c.Password) == 0 {
		return fmt.Errorf("password must be provided either directly or via a password file")
	}

	if len(c.Password) < 8 {
		Logger.Warn().
			Int("password_length", len(c.Password)).
			Msg("password is less than 8 characters, consider using a stronger password")
	}

	if !c.SSLMode {
		Logger.Warn().
			Bool("sslmode", c.SSLMode).
			Msg("ssl mode is disabled, consider enabling it for production environments or if exposing to outer network")
	}

	return nil
}
