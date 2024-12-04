# Examples

## Description

This repository provides various examples to help learn new concepts. Currently, it covers basic concepts including:
- golang-kafka
	- Producer and Consumer examples(sync and async producer)
	- Using to different library (sarama and confluent-kafka-go)
	- component-based architecture
	
- routeguide
	- Structure Data with protocol buffers.
	- Route Guide Package.
	- Server request with gRPC.
	- Server request with Http.
	- Structuring project using Hexagonal Architecture.

- hakernews
	- This project implements a **GraphQL API** using Go with a hexagonal architecture. It is designed to illustrate how to structure a project with input and output adapters, domains, services, and ports.
	- The API includes a simple example of handling GraphQL queries with dummy responses for educational purposes.

## Next steps
- routeguide
	- Transition the Hexagonal Architecture to a component-based architecture. This is because using Hexagonal Architecture can sometimes lead to mistakes. Since our repository is public, there’s a risk that users might bypass our services and access the repository directly, skipping the service layer.
	- Securete our services
	- Observe our system
	- Implement distributed services.
		- Server-to-server service discovery.
		- Coordinate our services with Consensus.
		- Discover servers and Load Balance From the Client.
		- Deploy Applications with Kubernetes Locally.
		- Deploy Applications with Kubernetes to the CLoud. 