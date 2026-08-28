package main

import (
	"log"
	"net/http"
	"os"

	payment "github.com/examples/go-distributed-payments-showcase/07-microservices/payment-svc/internal"
	"github.com/examples/go-distributed-payments-showcase/07-microservices/payment-svc/internal/memory"
)

func main() {
	addr  := env("ADDR", ":8080")
	store := memory.NewIdempotencyStore()
	repo  := memory.NewAccountRepo(map[string]int64{"acc-alice": 100_000, "acc-bob": 50_000})
	rsv   := memory.NewReservationStore()

	svc := payment.New(store, repo, rsv)
	h   := payment.NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	log.Printf("payment-svc listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
