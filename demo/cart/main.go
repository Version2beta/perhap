package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	perhap "github.com/version2beta/perhap/sdk/go"
)

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL // nats://127.0.0.1:4222
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("nats connect: %v", err)
	}
	defer nc.Close()
	log.Printf("connected to NATS at %s", natsURL)

	svc, err := perhap.NewService(nc, perhap.ServiceConfig[Cart]{
		Context:   "store",
		Domain:    "cart",
		Reducer:   CartReducer,
		Validator: CartValidator,
	})
	if err != nil {
		log.Fatalf("new service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("shutting down")
		cancel()
	}()

	if err := svc.Run(ctx); err != nil {
		log.Fatalf("service: %v", err)
	}
}
