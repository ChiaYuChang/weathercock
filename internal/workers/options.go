package workers

import (
	"fmt"
	"time"
)

// Options holds configurable parameters for the Runner.
type Options struct {
	Timeout          time.Duration
	HealthCheckPort  int
	HealthCheckHost  string
	ShutdownWaitTime time.Duration
}

// Option is a function type that modifies the Options struct.
type Option func(*Options) error

// WithTimeout sets the message processing timeout for the worker's Handle method.
func WithTimeout(d time.Duration) Option {
	return func(o *Options) error {
		if d < 0 {
			return fmt.Errorf("timeout should be positive: %v", d)
		}
		o.Timeout = d
		return nil
	}
}

// WithHealthCheckPort sets the listening port for the health check HTTP server.
func WithHealthCheckPort(port int) Option {
	return func(o *Options) error {
		if port <= 0 || port > 65535 {
			return fmt.Errorf("invalid port number: %d", port)
		}
		o.HealthCheckPort = port
		o.HealthCheckPort = port
		return nil
	}
}

func WithHealthCheckHost(host string) Option {
	return func(o *Options) error {
		o.HealthCheckHost = host
		return nil
	}
}

func WithShutdownWaitTime(d time.Duration) Option {
	return func(o *Options) error {
		if d < 0 {
			return fmt.Errorf("wait time should be positive: %v", d)
		}
		o.ShutdownWaitTime = d
		return nil
	}
}
