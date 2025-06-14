package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gcarrenho/component-based/option-1/cmd/app"
	"github.com/gcarrenho/component-based/option-1/internal/delivery/http/routes"
	"github.com/gcarrenho/component-based/option-1/internal/delivery/http/server"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}

	router := gin.Default()

	container := app.NewAppContainer()
	routes.Register(router, container)

	go server.Run(ctx, router, wg)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	cancel()
	wg.Wait()
}
