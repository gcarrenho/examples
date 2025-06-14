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

	"github.com/gcarrenho/hexagonal/bad-example/internal/adapters/in/handlers"
	"github.com/gcarrenho/hexagonal/bad-example/internal/adapters/out/repository"
	"github.com/gcarrenho/hexagonal/bad-example/internal/core/services"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}

	router := setupRouter()

	orderRepo := repository.NewOrdersRepository()
	ordersService := services.NewOrdersService(orderRepo)

	paymentsRepo := repository.NewPaymentsRepository()
	paymentsService := services.NewPaymentsService(paymentsRepo)

	routerGroup := router.Group("/")
	handlers.NewOrdersHandler(routerGroup, ordersService)
	handlers.NewPaymentsHandler(routerGroup, paymentsService)

	// This is the bad example, where we are using directly the repository in the handler
	// instead of using the service layer.
	handlers.NewPaymentsHandlerBad(routerGroup, paymentsRepo)

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
