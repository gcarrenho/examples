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
	"time"

	"github.com/redis/go-redis/v9"

	banking "github.com/examples/banking-core"
	worker "github.com/examples/payment-worker/internal"
	"github.com/examples/payment-worker/internal/fake"
	kafkaadapter "github.com/examples/payment-worker/internal/kafka"
	"github.com/examples/payment-worker/internal/ledger"
	"github.com/examples/payment-worker/internal/redisstore"
	"github.com/examples/payment-worker/internal/sepa"
	"github.com/examples/payment-worker/internal/swift"
)

func main() {
	brokers := strings.Split(env("KAFKA_BROKERS", "localhost:9092"), ",")
	redisURL := env("REDIS_URL", "redis://localhost:6379")
	pgURL := env("POSTGRES_URL", "postgres://payments:payments@localhost:5432/payments")
	logger := slog.Default()

	ropt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("redis url: %v", err)
	}
	idemStore := redisstore.New(redis.NewClient(ropt), 48*time.Hour)

	ledgerStore, err := ledger.NewStore(pgURL)
	if err != nil {
		log.Fatalf("ledger: %v", err)
	}
	defer ledgerStore.Close()

	sepaRail := sepa.New(env("EBA_URL", "https://step2.ebaclearing.eu"), "TESTBIC1", http.DefaultClient)
	swiftRail := swift.New(env("SWIFT_URL", "https://api.swift.com"), "TESTBIC1", http.DefaultClient)
	engine := banking.New(sepaRail, swiftRail, ledgerStore,
		banking.WithBackoff(banking.JitteredBackoff(5*time.Millisecond)),
		banking.WithRail(banking.RailFake, fake.New(100*time.Millisecond, logger)))

	resultsProducer, err := kafkaadapter.NewResultsProducer(brokers)
	if err != nil {
		log.Fatalf("kafka results producer: %v", err)
	}
	defer resultsProducer.Close()

	w := worker.New(engine, idemStore, resultsProducer, logger)
	consumer, err := kafkaadapter.NewConsumer(brokers, w, logger)
	if err != nil {
		log.Fatalf("kafka consumer: %v", err)
	}
	defer consumer.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("payment-worker consuming from Kafka %v", brokers)
	consumer.Run(ctx)
	log.Println("payment-worker stopped")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
