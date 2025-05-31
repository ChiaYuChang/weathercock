package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/ChiaYuChang/weathercock/pkgs/pb"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"
)

func main() {
	url := "nats://localhost:4222"
	nc, err := nats.Connect(
		url,
		nats.UserInfo(
			os.Getenv("NATS_USER"),
			os.Getenv("NATS_PASS"),
		),
	)
	if err != nil {
		log.Fatalf("Failed to connect to NATS server: %v", err)
	}

	defer func() {
		err := nc.Drain()
		if err != nil {
			log.Fatalf("Failed to drain NATS connection: %v", err)
		} else {
			log.Println("NATS connection drained successfully")
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	sub, err := nc.Subscribe(fmt.Sprintf("%s.>", global.NATSLogSubject), func(msg *nats.Msg) {
		level := msg.Subject[len(global.NATSLogSubject)+1:] // Extract level from subject
		log.Printf("[%s] Received log message: %s", strings.ToUpper(level), string(msg.Data))
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to log subject: %v", err)
		os.Exit(1)
	}
	log.Println("Press Ctrl+C to exit...")
	<-c
	if err := sub.Unsubscribe(); err != nil {
		log.Fatalf("Failed to unsubscribe from log subject: %v", err)
		os.Exit(1)
	}
	log.Printf("Unsubscribed from %s, exiting...\n", sub.Subject)
	// SimpleTextMessage demonstrates a simple text message exchange using NATS.
	// SimpleTextMessage(nc)

	// ProtobufMessage demonstrates a protobuf message exchange using NATS.
	// ProtobufMessage(nc)

	// LimitsBasedStreaming(nc)
}

func PullConsumers(nc *nats.Conn) {

}

func LimitsBasedStreaming(nc *nats.Conn) {
	PrintStreamState := func(ctx context.Context, stream jetstream.Stream) {
		info, err := stream.Info(ctx)
		if err != nil {
			log.Printf("Failed to get stream info: %v", err)
			return
		}
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal stream info: %v", err)
			return
		}
		log.Println("Stream Info:")
		log.Println(string(data))
	}
	log.Println("Creating a limits-based streaming example...")

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("Failed to create JetStream context: %v", err)
	}

	// one stream can have multiple subjects, but they must not overlap.
	// no two streams can have overlapping subjects; otherwise, the message will be
	// presisted twice.
	scfg := jetstream.StreamConfig{
		Name:     "BROWSER_EVENTS", // Commonly uppercase for stream names
		Subjects: []string{"page-events.>"},
	}

	// in-momory storage and file storage are supported.
	scfg.Storage = jetstream.FileStorage

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := js.CreateStream(ctx, scfg)
	if err != nil {
		log.Fatalf("Failed to create stream: %v", err)
	}
	log.Printf("Stream created: %s, Subjects: %v", scfg.Name, scfg.Subjects)

	for _, event := range []string{
		"page-events.page_loaded",
		"page-events.button_clicked",
		"page-events.form_submitted",
		"page-events.mouse_moved",
		"page-events.key_pressed",
		"page-events.scroll_event",
		"page-events.touch_gesture",
	} {
		js.Publish(ctx, event, nil)
	}

	for _, event := range []string{
		"page-events.page_loaded",
		"page-events.button_clicked",
		"page-events.form_submitted",
		"page-events.mouse_moved",
		"page-events.key_pressed",
		"page-events.scroll_event",
		"page-events.touch_gesture",
	} {
		js.PublishAsync(event, nil)
	}

	select {
	case <-js.PublishAsyncComplete():
		log.Println("All async messages published successfully")
	case <-time.After(5 * time.Second):
		log.Println("Timed out waiting for async messages to be published")
	}

	PrintStreamState(ctx, stream)

	scfg.MaxMsgs = 10
	js.UpdateStream(ctx, scfg)
	log.Printf("Set max messages to %d for stream %s", scfg.MaxMsgs, scfg.Name)
	PrintStreamState(ctx, stream)

	scfg.MaxBytes = 300
	js.UpdateStream(ctx, scfg)
	log.Printf("Set max bytes to %d for stream %s", scfg.MaxBytes, scfg.Name)
	PrintStreamState(ctx, stream)

	scfg.MaxAge = 1 * time.Second
	js.UpdateStream(ctx, scfg)
	log.Printf("Set max age to %s for stream %s", scfg.MaxAge, scfg.Name)
	PrintStreamState(ctx, stream)

	sleep := rand.IntN(5) + 3
	log.Printf("Sleeping for %d seconds to allow messages to be processed...", sleep)
	time.Sleep(time.Duration(sleep) * time.Second)

	PrintStreamState(ctx, stream)

	ccfg := jetstream.ConsumerConfig{
		Name:        "PAGE_EVENTS_CONSUMER",
		Description: "An example consumer for page events",
	}

	consumer, err := js.CreateConsumer(ctx, scfg.Name, ccfg)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	log.Printf("Consumer created: %s", ccfg.Name)
	cContent, err := consumer.Consume(func(msg jetstream.Msg) {
		log.Printf("Received message on subject %s: %s", msg.Subject(), string(msg.Data()))
		if err := msg.Ack(); err != nil {
			log.Printf("Failed to acknowledge message: %v", err)
		} else {
			log.Printf("Message acknowledged: %s", msg.Subject())
		}
	})
	if err != nil {
		log.Fatalf("Failed to consume messages: %v", err)
	}

	data, _ := json.MarshalIndent(cContent, "", "  ")
	log.Printf("Consumer content: %s", string(data))
}

func SimpleTextMessage(nc *nats.Conn) {
	log.Println("Sending a simple text message...")
	subj := "greet.*"
	sub, err := nc.Subscribe(subj, func(msg *nats.Msg) {
		log.Printf("Received message on subject %s: %s", msg.Subject, string(msg.Data))
		name := msg.Subject[len("greet."):] // Extract name from subject
		msg.Respond([]byte(fmt.Sprintf("Hello, %s! Nice to meet you!", name)))
	})

	if err != nil {
		log.Fatalf("Failed to subscribe to subject %s: %v", subj, err)
	}
	log.Printf("Subscribed to subject %s", subj)

	for _, name := range []string{"alice", "bob", "joe"} {
		msg := fmt.Sprintf("greet.%s", name)
		log.Printf("Requesting greeting for %s", msg)
		// Send a request to the subject
		if rep, err := nc.Request(msg, nil, nats.DefaultTimeout); err != nil {
			log.Printf("Failed to request greeting for %s: %v", name, err)
		} else {
			log.Printf("Received response for %s: %s", name, string(rep.Data))
		}
		time.Sleep(3 * time.Second) // Sleep to allow processing
	}

	if err := sub.Unsubscribe(); err != nil {
		log.Fatalf("Failed to unsubscribe from subject %s: %v", subj, err)
	} else {
		log.Printf("Unsubscribed from subject %s", subj)
	}

	_, err = nc.Request("greet.alice", nil, nats.DefaultTimeout)
	fmt.Printf("Request to greet.alice: %v\n", err)
}

func ProtobufMessage(nc *nats.Conn) {
	log.Println("Sending a protobuf message...")

	subj := "greet.*"
	_, err := nc.Subscribe(subj, func(msg *nats.Msg) {
		log.Printf("Received message on subject %s", msg.Subject)
		var req pb.GreetRequest
		proto.Unmarshal(msg.Data, &req)

		rep := pb.GreetResponse{
			Message: fmt.Sprintf("Hello, %s! Nice to meet you!", req.Name),
		}
		data, _ := proto.Marshal(&rep)
		msg.Respond(data)
	})

	if err != nil {
		log.Fatalf("Failed to subscribe to subject %s: %v", subj, err)
	}
	log.Printf("Subscribed to subject %s", subj)

	for _, name := range []string{"alice", "bob", "joe"} {
		req := &pb.GreetRequest{Name: name}
		data, err := proto.Marshal(req)
		if err != nil {
			log.Fatalf("Failed to marshal request: %v", err)
		}

		log.Printf("Requesting greeting for %s", name)
		if rep, err := nc.Request(fmt.Sprintf("greet.%s", name), data, nats.DefaultTimeout); err != nil {
			log.Printf("Failed to request greeting for %s: %v", name, err)
		} else {
			var res pb.GreetResponse
			if err := proto.Unmarshal(rep.Data, &res); err != nil {
				log.Printf("Failed to unmarshal response: %v", err)
			} else {
				log.Printf("Received response for %s: %s", name, res.Message)
			}
		}
	}
}
