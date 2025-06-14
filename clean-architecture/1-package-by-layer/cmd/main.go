package main

import (
	"fmt"

	"github.com/gcarrenho/package-by-layer/handlers"
)

func main() {
	fmt.Println("Running server...")
	handlers.SetupRoutes()
}
