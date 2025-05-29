package global

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
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

// MarshalJSON is a custom JSON marshaller that masks the password field.
func (c PostgresConfig) MarshalJSON() ([]byte, error) {
	password := c.Password
	if c.PasswordFile != "" {
		password = strings.Repeat("*", len(c.Password)+rand.IntN(5))
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

	c.Password = strings.TrimSpace(string(data))
	return nil
}

// ConnectionString returns the PostgreSQL connection string based on the configuration.
func (c *PostgresConfig) ConnectionString() string {
	sslmode := "disable"
	if c.SSLMode {
		sslmode = "enable"
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Username, c.Password, c.Host, c.Port, c.Database, sslmode)
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
func (c *PostgresConfig) Validate() error {
	v := Validate()

	if err := v.Struct(c); err != nil {
		return fmt.Errorf("invalid Postgres configuration: %w", err)
	}

	if c.Password == "" {
		Logger.Warn().
			Str("password_file", c.PasswordFile).
			Msg("password is empty, trying to read from password file")
		if err := c.ReadPasswordFile(); err != nil {
			return err
		}
		Logger.Warn().
			Msg("call .ReadPasswordFile() before Validate() to remove this warning")
	}

	if len(c.Password) == 0 {
		return fmt.Errorf("password must be provided either directly or via a password file")
	}

	if len(c.Password) < 8 {
		Logger.Warn().
			Int("password_length", len(c.Password)).
			Msg("password is less than 8 characters, consider using a stronger password")
	}

	return nil
}
