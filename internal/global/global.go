package global

import (
	"os"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var Logger zerolog.Logger

var validate struct {
	*validator.Validate
	sync.Once
}

func Validate() *validator.Validate {
	validate.Once.Do(func() {
		validate.Validate = validator.New()
	})
	return validate.Validate
}

func Initialization() {
	nc, err := nats.Connect(
		nats.DefaultURL,
		nats.UserInfo(
			os.Getenv("NATS_USER"),
			os.Getenv("NATS_PASS"),
		),
	)
	if err != nil {
		panic("Failed to connect to NATS server: " + err.Error())
	}

	Logger = log.Output(zerolog.MultiLevelWriter(
		zerolog.ConsoleWriter{Out: os.Stdout},
		&NatsLogWriter{
			Conn:    nc,
			Subject: NATSLogSubject,
		},
	))

	// initialize validator
	_ = Validate()

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func LoadAPIConfig(filename, filetype, filepath string) error {
	viper.SetConfigName(filename)
	viper.SetConfigType(filetype)
	viper.AddConfigPath(filepath)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	return nil
}
