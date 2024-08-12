# Golang-Kafka
Apache Kafka is a distributed event streaming platform that provides a reliable and scalable way to publish, subscribe, and process streams of records in real-time.
Golang, with its simplicity, efficiency, and strong concurrency support, is an excellent choice for building high-performance applications that integrate with Kafka

## Description

This repository provides an examples to help learn kafka basic concepts. Currently, it have three services:

- order-service.
	- it will publish a message to the Kafka topic and be consumed by another service that listens to or subscribes to it. 
- product-service.
 	- It will consumed the message to the topic. Consuming the messages that arrive after the subscription.
- user-service.
	- It will consumed the message to the topic. Consuming the messages that exist before the subscription.

In order to learn the use of two library (sarama and confluent-kafka-go), we created order-service using sarama, who is the owner to produce msg to the topic.
And on other two services we use confluent-kafka-go to consume the msg to the topic.

## Setup
- Kafka installed and running on your local or remote environment.
- Installed GO

### Using JVM Based Apache Kafka Docker Image

Get the Docker image:

	docker pull apache/kafka:3.8.0 

Start the Kafka Docker container:

	docker run -p 9092:9092 apache/kafka:3.8.0

## Run Services
Move to inside the server that you want to run.

order-service

	go run order_server.go

product-service

	go run product_server.go

user-service

	go run user_server.go