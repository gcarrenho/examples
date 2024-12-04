package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gcarrenho/hackernews/internal/adapters/in/graph"
	"github.com/gcarrenho/hackernews/internal/adapters/in/graph/resolvers"
	"github.com/gcarrenho/hackernews/internal/adapters/out/db/mysql"
	"github.com/gcarrenho/hackernews/internal/core/services"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	linkRepository := mysql.NewLinkRepository()
	linkSrv := services.NewLinkSrv(linkRepository)

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &resolvers.Resolver{
		LinkSrv: linkSrv,
	}}))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
