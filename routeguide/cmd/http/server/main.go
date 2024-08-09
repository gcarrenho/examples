package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	hdl "github.com/gcarrenho/routeguide/internal/adapters/in/http"
	"github.com/gcarrenho/routeguide/internal/adapters/out/repository"
	service "github.com/gcarrenho/routeguide/internal/core/services"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

func main() {
	flag.Parse()
	repo, err := repository.NewFeatureRepository("")
	if err != nil {
		log.Fatalf("Failed to create feature repository: %v", err)
	}
	featureSvc := service.NewFeatureService(repo)

	router := setupRouter()

	routerGroup := router.Group("routeguide/")
	hdl.NewRouteGUideHdl(routerGroup, featureSvc)

	runServer(router)
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

func runServer(router *gin.Engine) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	ginSvc := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		runErr := ginSvc.ListenAndServe()
		if runErr != nil && !errors.Is(runErr, http.ErrServerClosed) {
			log.Fatal("could no start")
		}
	}()
	<-ctx.Done()
	stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if shutdownErr := ginSvc.Shutdown(ctx); shutdownErr != nil {
		log.Fatal("shutdown")
	}
}
