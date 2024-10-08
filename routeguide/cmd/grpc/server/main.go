package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	adapters "github.com/gcarrenho/routeguide/internal/adapters/in/grpc"
	"github.com/gcarrenho/routeguide/internal/adapters/out/repository"
	service "github.com/gcarrenho/routeguide/internal/core/services"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	tls        = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile   = flag.String("cert_file", "path/to/server.crt", "The file containing the CA root certificate")
	keyFile    = flag.String("key_file", "path/to/server.key", "The file containing the server's private key")
	port       = flag.Int("port", 50051, "The server port")
	jsonDBFile = flag.String("json_db_file", "", "A json file containing a list of features")
)

func main() {
	flag.Parse()
	repo, err := repository.NewFeatureRepository(*jsonDBFile)
	if err != nil {
		log.Fatalf("Failed to create feature repository: %v", err)
	}
	featureSvc := service.NewFeatureService(repo)

	var opts []grpc.ServerOption
	if *tls {
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("failed to generate credentials: %v", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	gRPCServer, err := adapters.NewGRPCServer(&adapters.Config{FeatureSvc: featureSvc}, opts...)
	if err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	grpcLn, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	if err := gRPCServer.Serve(grpcLn); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
