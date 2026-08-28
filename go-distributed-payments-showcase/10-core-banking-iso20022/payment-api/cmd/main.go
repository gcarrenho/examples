package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	api "github.com/examples/payment-api/internal"
	"github.com/examples/payment-api/internal/kafka"
)

func main() {
	addr := env("ADDR", ":8083")
	brokers := strings.Split(env("KAFKA_BROKERS", "localhost:9092"), ",")
	redisURL := env("REDIS_URL", "redis://localhost:6379")
	businessTimeout := envDuration("PAYMENT_TIMEOUT", 5*time.Minute)
	logger := slog.Default()

	ropt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("redis url: %v", err)
	}
	rdb := redis.NewClient(ropt)

	results := api.NewResultsStore(rdb, 30*24*time.Hour)
	pending := api.NewPendingIndex(rdb)
	callbacks := api.NewCallbackStore(rdb, 2*businessTimeout)
	webhook := api.NewWebhookNotifier(http.DefaultClient, logger)

	producer, err := kafka.NewProducer(brokers)
	if err != nil {
		log.Fatalf("kafka producer: %v", err)
	}
	defer producer.Close()

	processor := api.NewResultProcessor(results, pending, callbacks, webhook, logger)
	resultsConsumer, err := kafka.NewResultsConsumer(brokers, processor, logger)
	if err != nil {
		log.Fatalf("kafka results consumer: %v", err)
	}
	defer resultsConsumer.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go resultsConsumer.Run(ctx) // populates ResultsStore in the background for GET /payments/{uetr}

	reaper := api.NewTimeoutReaper(pending, results, callbacks, webhook, businessTimeout, logger)
	go reaper.Run(ctx, 30*time.Second) // sweeps for orders stuck PDNG past the business SLA

	h := api.NewHandler(producer, results, pending, callbacks)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	logger.Info("payment-api listening", slog.String("addr", addr), slog.Any("brokers", brokers),
		slog.Duration("business_timeout", businessTimeout))
	log.Fatal(http.ListenAndServe(addr, mux))
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return fallback
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
