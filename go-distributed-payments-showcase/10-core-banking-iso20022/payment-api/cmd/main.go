package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	api "github.com/examples/payment-api/internal"
	"github.com/examples/payment-api/internal/kafka"
)

func main() {
	addr := env("ADDR", ":8083")
	brokers := strings.Split(env("KAFKA_BROKERS", "localhost:9092"), ",")
	logger := slog.Default()

	producer, err := kafka.NewProducer(brokers)
	if err != nil {
		log.Fatalf("kafka producer: %v", err)
	}
	defer producer.Close()

	results := api.NewResultsStore()
	resultsConsumer, err := kafka.NewResultsConsumer(brokers, results, logger)
	if err != nil {
		log.Fatalf("kafka results consumer: %v", err)
	}
	defer resultsConsumer.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go resultsConsumer.Run(ctx) // populates ResultsStore in the background for GET /payments/{uetr}

	h := api.NewHandler(producer, results)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	logger.Info("payment-api listening", slog.String("addr", addr), slog.Any("brokers", brokers))
	log.Fatal(http.ListenAndServe(addr, mux))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
