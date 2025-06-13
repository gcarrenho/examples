package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"

	ordersHdl "github.com/gcarrenho/hexagonal/good-example/internal/orders/adapters/in/handlers"
	paymentsHdl "github.com/gcarrenho/hexagonal/good-example/internal/payments/adapters/in/handlers"

	ordersRepo "github.com/gcarrenho/hexagonal/good-example/internal/orders/adapters/out/repository"
	paymentsRepo "github.com/gcarrenho/hexagonal/good-example/internal/payments/adapters/out/repository"

	ordersSvc "github.com/gcarrenho/hexagonal/good-example/internal/orders/core/services"
	paymentsSvc "github.com/gcarrenho/hexagonal/good-example/internal/payments/core/services"

	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}

	router := setupRouter()

	orderRepo := ordersRepo.NewOrdersRepository()
	ordersService := ordersSvc.NewOrdersService(orderRepo)

	paymentsRepo := paymentsRepo.NewPaymentsRepository()
	paymentsService := paymentsSvc.NewPaymentsService(paymentsRepo)

	routerGroup := router.Group("/")
	ordersHdl.NewOrdersHandler(routerGroup, ordersService)
	paymentsHdl.NewPaymentsHandler(routerGroup, paymentsService)

	go runServer(ctx, router, wg)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	cancel()
	wg.Wait()
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(
		requestid.New(),
		gin.Recovery(),
	)
	return router
}

func runServer(ctx context.Context, router *gin.Engine, wg *sync.WaitGroup) {
	log.Info().Msg("Starting HTTP server...")

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start server
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Info().Msg("Shutdown initiated")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-shutdownCtx.Done():
		log.Warn().Msg("Timeout waiting for goroutines to finish")
	}

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Forced shutdown")
	}

	log.Info().Msg("Server gracefully stopped")
}
