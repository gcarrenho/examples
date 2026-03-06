// Package main
//
// This program demonstrates the use of a `sync.WaitGroup` to coordinate the execution
// of goroutines in Go. It highlights the concept of a "join point," where the main
// goroutine waits for other goroutines to complete before proceeding.
//
// Key Concepts:
// 1. **WaitGroup Usage**:
//   - The `sync.WaitGroup` is used to track the number of goroutines that need to complete.
//     In this example, the `wg.Add(1)` call increments the counter before launching a goroutine,
//     and the `defer wg.Done()` call decrements the counter once the goroutine finishes execution.
//
// 2. **Join Point**:
//   - The `wg.Wait()` call serves as the "join point," where the main goroutine blocks
//     until the counter in the WaitGroup reaches zero, ensuring all launched goroutines
//     have completed their work.
//
// 3. **Deferred Cleanup**:
//   - The use of `defer wg.Done()` ensures that the counter is decremented even if the
//     goroutine encounters an error or exits early.
//
// Takeaway:
//   - This example provides a minimal illustration of how `sync.WaitGroup` can be used
//     to manage concurrency in Go, ensuring that the main goroutine waits for all
//     spawned goroutines to finish their tasks.
package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	sayHello := func() {
		defer wg.Done()
		fmt.Println("hello")
	}
	wg.Add(1)
	go sayHello()
	wg.Wait() // <1>
}

// 1. This is the join point
