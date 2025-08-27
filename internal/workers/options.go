package workers

import "time"

// Options holds configurable parameters for the Runner.
type Options struct {
	Timeout         time.Duration
	HealthCheckPort int
	HealthCheckHost string
}

// Option is a function type that modifies the Options struct.
type Option func(*Options) error

// WithTimeout sets the message processing timeout for the worker's Handle method.
func WithTimeout(t time.Duration) Option {
	return func(o *Options) error {
		o.Timeout = t
		return nil
	}
}

// WithHealthCheckPort sets the listening port for the health check HTTP server.
func WithHealthCheckPort(port int) Option {
	return func(o *Options) error {
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
