package main

import (
	"context"
	"log"
	"net/http"
	"os"

	gateway "github.com/examples/go-distributed-payments-showcase/08-acquirer-gateway/internal"
)

// localGateway aprueba cualquier request — solo para dev/testing local.
// En producción: prisma.New(PRISMA_URL, client) y adyen.New(ADYEN_URL, ADYEN_KEY, merchantID, client).
type localGateway struct{ name string }

func (g *localGateway) Authorize(_ context.Context, req gateway.AuthRequest) (gateway.AuthResponse, error) {
	return gateway.AuthResponse{
		Approved:   true,
		AuthCode:   "LOCAL01",
		NetworkRef: "REF-" + g.name + "-" + req.IdempotencyKey,
	}, nil
}

func main() {
	addr := env("ADDR", ":8083")

	ar := gateway.WithCircuitBreaker(&localGateway{"prisma-AR"}, 5)
	eu := gateway.WithCircuitBreaker(&localGateway{"adyen-EU"}, 5)
	router := gateway.NewRouter(eu, map[string]gateway.Gateway{
		"AR": ar,
		"BR": gateway.WithCircuitBreaker(&localGateway{"cielo-BR"}, 5),
		"GB": gateway.WithCircuitBreaker(&localGateway{"worldpay-GB"}, 5),
	})

	mux := http.NewServeMux()
	gateway.NewHandler(router).RegisterRoutes(mux)
	log.Printf("acquirer-gateway listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
