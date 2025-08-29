package global

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/nats-io/nats.go"
	"github.com/spf13/viper"
)

const (
	NATSLogStream        = "weathercock_logs"
	NATSLogStreamSubject = "weathercock.logs.>"
	NATSTaskStream       = "weathercock_tasks"
)

// NATSConfig holds configuration for connecting to a NATS server.
// Authentication by username and password.
type NATSConfig struct {
	Host      string `json:"host"     validate:"required"                  mapstructure:"host"`
	Port      int    `json:"port"     validate:"required"                  mapstructure:"port"`
	Username  string `json:"username" validate:"required_without=Token"    mapstructure:"username"`
	Password  string `json:"password" validate:"required_without=Token"    mapstructure:"password"`
	Token     string `json:"token"    validate:"required_without=Password" mapstructure:"token"`
	JetStream bool   `json:"jet_stream"                                    mapstructure:"jet_stream"`
}

// DefaultNATSConfig returns a default NATSConfig.
func DefaultNATSConfig() *NATSConfig {
	return &NATSConfig{
		Host:     "localhost",
		Port:     4222,
		Username: "default",
		Password: "",
		Token:    "",
	}
}

// LoadNATSConfig returns the NATS configuration from Viper.
func LoadNATSConfig() *NATSConfig {
	conf := DefaultNATSConfig()
	conf.Host = utils.DefaultIfZero(viper.GetString("NATS_HOST"), conf.Host)
	conf.Port = utils.DefaultIfZero(viper.GetInt("NATS_CLI_PORT"), conf.Port)
	conf.Username = utils.DefaultIfZero(viper.GetString("NATS_USER"), conf.Username)
	conf.Password = utils.DefaultIfZero(viper.GetString("NATS_PASS"), conf.Password)
	conf.JetStream = viper.GetBool("NATS_JETSTREAM")
	return conf
}

// MarshalJSON is a custom JSON marshaller that masks the password field.
func (c NATSConfig) MarshalJSON() ([]byte, error) {
	password := c.Password
	if c.Password != "" {
		password = utils.Mask(password)
	}

	type Alias NATSConfig
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
func (c NATSConfig) MarshalJSONPlain() ([]byte, error) {
	type Alias NATSConfig
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(&c),
	})
}

// URL returns the NATS connection string based on the configuration.
func (c NATSConfig) URL() string {
	return fmt.Sprintf("nats://%s:%d", c.Host, c.Port)
}

// URLString returns the NATS connection string based on the configuration.
// It masks the password in the connection string.
func (c NATSConfig) URLString() string {
	password := c.Password
	if password != "" {
		password = utils.Mask(password)
	}
	return fmt.Sprintf("nats://%s:%s@%s:%d", c.Username, password, c.Host, c.Port)
}

// Connect establishes a connection to the NATS server.
func (c NATSConfig) Connect() (*nats.Conn, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("invalid NATS configuration: %w", err)
	}

	if c.Token != "" {
		return nats.Connect(c.URL(), nats.Token(c.Token))
	}
	return nats.Connect(c.URL(), nats.UserInfo(c.Username, c.Password))
}

// ConnectJetStream establishes a connection to the NATS server and initializes JetStream.
// It also creates the necessary streams if they don't exist.
func (c NATSConfig) ConnectJetStream() (*nats.Conn, nats.JetStreamContext, error) {
	nc, err := c.Connect()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// Create weathercock_logs stream
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     NATSLogStream,
		Subjects: []string{NATSLogStreamSubject},
		MaxMsgs:  -1,
		MaxBytes: -1,
		MaxAge:   0,
		Storage:  nats.FileStorage,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to add weathercock_logs stream: %w", err)
	}

	// Create weathercock_tasks stream
	_, err = js.AddStream(&nats.StreamConfig{
		Name: NATSTaskStream,
		Subjects: []string{
			"task.create",
			"task.scrape",
			"task.generate_title",
			"task.extract.keyword",
		},
		MaxMsgs:  -1,
		MaxBytes: -1,
		MaxAge:   0,
		Storage:  nats.FileStorage,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to add weathercock_tasks stream: %w", err)
	}

	return nc, js, nil
}

// String returns a string representation of the NATSConfig.
// It masks the password in the string representation.
func (c NATSConfig) String() string {
	sb := strings.Builder{}
	sb.WriteString("NATS Configuration:\n")
	sb.WriteString(fmt.Sprintf("Host: %s\n", c.Host))
	sb.WriteString(fmt.Sprintf("Port: %d\n", c.Port))
	sb.WriteString(fmt.Sprintf("Username: %s\n", c.Username))
	sb.WriteString(fmt.Sprintf("Password: %s\n", utils.Mask(c.Password)))
	return sb.String()
}

// Validate checks the NATSConfig for required fields and conditions.
func (c NATSConfig) Validate() error {
	err := Validator.Struct(c)
	if err != nil {
		return fmt.Errorf("NATS config validation failed: %w", err)
	}
	return nil
}

type NATSLogWriter struct {
	Js      nats.JetStreamContext
	Subject string
}

func (w *NATSLogWriter) extractLevel(s string) string {
	re := regexp.MustCompile(`"level"\s*:\s*"(\w+)"`)
	matches := re.FindStringSubmatch(s)
	if len(matches) < 2 {
		return "unknown"
	}
	return matches[1]
}

func (w *NATSLogWriter) Write(p []byte) (n int, err error) {
	level := w.extractLevel(string(p))

	if w.Js == nil {
		return 0, fmt.Errorf("JetStream context is nil")
	}

	if _, err := w.Js.Publish(fmt.Sprintf("%s.%s", NATSLogStreamSubject, level), p); err != nil {
		return 0, err
	}

	return len(p), nil
}
