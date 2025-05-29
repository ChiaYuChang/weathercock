package global

type NatsConfig struct {
	Host string `json:"host"     validate:"required" mapstructure:"host"`
	Port int    `json:"port"     validate:"required" mapstructure:"port"`
}
