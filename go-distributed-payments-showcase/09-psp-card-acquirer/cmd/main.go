package main

import (
	"context"
	"log"
	"net/http"
	"os"

	authorization "github.com/examples/go-distributed-payments-showcase/09-psp-card-acquirer/internal"
)

// localNetwork aprueba cualquier tarjeta — solo para dev/testing local.
// En producción cada brand recibe su red real: visanet.New(...), mastercard.New(...), amex.New(...).
type localNetwork struct{ name string }

func (n *localNetwork) Authorize(_ context.Context, txn authorization.CardTransaction) (authorization.CardTransaction, error) {
	txn.Status = authorization.StatusApproved
	txn.AuthCode = "LOCAL01"
	txn.NetworkRef = "STUB-REF-" + n.name
	return txn, nil
}

func main() {
	addr := env("ADDR", ":8084")

	stub := &localNetwork{"stub"}
	router := authorization.NewNetworkRouter(stub).
		Register(authorization.BrandVisa, stub).
		Register(authorization.BrandMastercard, stub).
		Register(authorization.BrandAmex, stub)

	svc := authorization.NewService(router, authorization.NewMemoryIdempotency())
	h := authorization.NewHandler(svc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	log.Printf("psp-card-acquirer listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
