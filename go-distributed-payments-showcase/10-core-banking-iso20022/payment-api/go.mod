module github.com/examples/payment-api

go 1.26

require (
	github.com/IBM/sarama v1.43.0
	github.com/examples/banking-core v0.0.0-00010101000000-000000000000
)

replace github.com/examples/banking-core => ../banking-core
