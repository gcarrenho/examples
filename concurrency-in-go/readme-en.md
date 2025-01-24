# Concurrency in Go

This repository, titled `currency-in-go`, is dedicated to exploring **concurrency concepts** and **patterns in Go**. It aims to provide a comprehensive understanding of how Go handles concurrency, inspired by principles from Communicating Sequential Processes (CSP) and the philosophy of the Go programming language.

---

## Overview

Concurrency in Go is not about parallelism but about structuring applications to handle multiple tasks simultaneously. This repository explains:
- Core concepts of concurrency in Go.
- Common concurrency patterns.
- Best practices for writing efficient and safe concurrent programs.

---

## Key Concepts

1. **Goroutines**  
   Goroutines are lightweight threads managed by the Go runtime, allowing efficient execution of concurrent tasks.

2. **Channels**  
   Channels enable communication and synchronization between goroutines in a thread-safe manner, following Go’s mantra:  
   *“Do not communicate by sharing memory; instead, share memory by communicating.”*

3. **Synchronization Primitives**  
   Tools like `sync.Mutex` and `sync.WaitGroup` provide low-level control for cases where channels might not be ideal.

4. **Concurrency vs Parallelism**  
   - Concurrency: Dealing with multiple tasks at once.
   - Parallelism: Executing multiple tasks at the same time.

---

## Concurrency Patterns

This repository includes practical implementations of the following patterns:

1. **Fan-out/Fan-in**  
   Efficiently distribute tasks to multiple workers and aggregate results.

2. **Pipeline**  
   Create a series of stages where each stage processes data and passes it to the next.

3. **Worker Pool**  
   Manage a fixed number of workers processing jobs concurrently.

4. **Select Statements**  
   Handle multiple channels for communication between goroutines.

---

## How to Use This Repository

- Explore the provided code examples to learn about concurrency concepts and patterns.
- Use these patterns as a foundation to design scalable and efficient concurrent systems.
- Experiment with the examples and modify them to fit your specific use cases.

---

## Resources

- [Go Documentation](https://golang.org/doc/)
- [Concurrency in Go by Katherine Cox-Buday](https://www.oreilly.com/library/view/concurrency-in-go/9781491941294/)

---

## Contributing

Feel free to contribute with additional examples, explanations, or improvements to the existing patterns!