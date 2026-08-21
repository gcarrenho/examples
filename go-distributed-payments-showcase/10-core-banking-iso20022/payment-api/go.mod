module github.com/examples/payment-api

go 1.26

require (
	github.com/IBM/sarama v1.43.0
	github.com/examples/banking-core v0.0.0-00010101000000-000000000000
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/redis/go-redis/v9 v9.5.1 // indirect
)

replace github.com/examples/banking-core => ../banking-core
