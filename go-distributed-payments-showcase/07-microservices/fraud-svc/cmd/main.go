package main

import (
	"log"
	"net/http"
	"os"
	"time"

	fraud "github.com/examples/go-distributed-payments-showcase/07-microservices/fraud-svc/internal"
	"github.com/examples/go-distributed-payments-showcase/07-microservices/fraud-svc/internal/payment"
)

func main() {
	addr       := env("ADDR", ":8082")
	paymentURL := env("PAYMENT_SVC_URL", "http://localhost:8080")

	// The client satisfies fraud's paymentGateway{Charge+Reserve} via structural typing.
	// payment-svc never knows about this interface — the contract is the HTTP API.
	paymentClient := payment.NewClient(paymentURL, &http.Client{Timeout: 10 * time.Second})
	svc           := fraud.New(paymentClient)
	h             := fraud.NewHandler(svc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	log.Printf("fraud-svc listening on %s (payment-svc at %s)", addr, paymentURL)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
