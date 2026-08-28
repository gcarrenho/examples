package main

import (
	"log"
	"net/http"
	"os"
	"time"

	orders "github.com/examples/go-distributed-payments-showcase/07-microservices/orders-svc/internal"
	"github.com/examples/go-distributed-payments-showcase/07-microservices/orders-svc/internal/payment"
)

func main() {
addr       := env("ADDR", ":8081")
paymentURL := env("PAYMENT_SVC_URL", "http://localhost:8080")
paymentClient := payment.NewHTTPClient(paymentURL, &http.Client{Timeout: 10 * time.Second})
svc := orders.New(paymentClient)
h := orders.NewHandler(svc)
mux := http.NewServeMux()
h.RegisterRoutes(mux)
log.Printf("orders-svc listening on %s", addr)
log.Fatal(http.ListenAndServe(addr, mux))
}

func env(key, fallback string) string {
if v := os.Getenv(key); v != "" {
return v
}
return fallback
}
