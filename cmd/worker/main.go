package main

// import (
// 	"context"
// 	"os"
// 	"os/signal"
// 	"syscall"

// 	"github.com/ChiaYuChang/weathercock/internal/global"
// 	"github.com/ChiaYuChang/weathercock/internal/nats"
// )

// func main() {
// 	if err := global.L
// 	oadConfigs(".env", "env", []string{"."}); err != nil {
// 		global.Logger.Fatal().Err(err).Msg("Failed to load configurations")
// 	}

// 	// Initialize NATS connection
// 	natsConn := global.NATS()
// 	if natsConn.Conn == nil {
// 		global.Logger.Fatal().Msg("NATS connection is nil")
// 	}
// 	defer func() {
// 		if natsConn.Conn != nil {
// 			natsConn.Conn.Close()
// 			global.Logger.Info().Msg("NATS connection closed")
// 		}
// 	}()

// 	if !global.Config().NATS.JetStream {
// 		global.Logger.Fatal().Msg("JetStream is not enabled in NATS configuration. Worker requires JetStream.")
// 	}

// 	// Create a new NATS consumer
// 	consumer := nats.NewConsumer(natsConn.Js, global.Logger)

// 	// Run all consumers
// 	subscriptions, err := consumer.RunConsumers(context.Background())
// 	if err != nil {
// 		global.Logger.Fatal().Err(err).Msg("Failed to run NATS consumers")
// 	}
// 	defer nats.CloseSubscriptions(subscriptions, global.Logger)

// 	global.Logger.Info().Msg("NATS worker started. Waiting for messages...")

// 	// Graceful shutdown
// 	stop := make(chan os.Signal, 1)
// 	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
// 	<-stop

// 	global.Logger.Info().Msg("Shutting down NATS worker...")
// }
