package global

import (
	"fmt"
	"regexp"

	"github.com/nats-io/nats.go"
)

const NATSLogSubject = "weathercock.logs"

type NatsConfig struct {
	Host string `json:"host"     validate:"required" mapstructure:"host"`
	Port int    `json:"port"     validate:"required" mapstructure:"port"`
}

func (c *NatsConfig) URL() string {
	return fmt.Sprintf("nats://%s:%d", c.Host, c.Port)
}

type NatsLogWriter struct {
	Conn    *nats.Conn
	Subject string
}

func (w *NatsLogWriter) extractLevel(s string) string {
	re := regexp.MustCompile(`"level"\s*:\s*"(\w+)"`)
	matches := re.FindStringSubmatch(s)
	if len(matches) < 2 {
		return "unknown"
	}
	return matches[1]
}

func (w *NatsLogWriter) Write(p []byte) (n int, err error) {
	level := w.extractLevel(string(p))

	if w.Conn == nil {
		return 0, nats.ErrConnectionClosed
	}

	if err := w.Conn.Publish(fmt.Sprintf("%s.%s", NATSLogSubject, level), p); err != nil {
		return 0, err
	}

	return len(p), nil
}

// implements the io.Reader interface
type NatsReader struct {
	Conn    *nats.Conn
	Subject string
}

func (r *NatsReader) Read(p []byte) (n int, err error) {
	if r.Conn == nil {
		return 0, nats.ErrConnectionClosed
	}

	msg, err := r.Conn.Request(r.Subject, nil, nats.DefaultTimeout)
	if err != nil {
		return 0, err
	}

	copy(p, msg.Data)
	return len(msg.Data), nil
}
