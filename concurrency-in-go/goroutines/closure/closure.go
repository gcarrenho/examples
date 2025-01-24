// Package main
//
// This program demonstrates the concept of closures in Go,
// particularly how variables captured by a goroutine share memory
// with the main goroutine. The `salutation` variable is shared
// between the main goroutine and a spawned goroutine, illustrating
// how concurrency can introduce data races or unexpected behavior
// if not managed carefully.
//
// Key Concepts:
// - Closures in Go and their interaction with goroutines.
// - Shared memory model in Go: goroutines execute in the same address space.
// - Proper synchronization of shared variables using sync.WaitGroup.
//
// The example highlights the importance of understanding the shared
// nature of variables in goroutines and the potential pitfalls of
// concurrent programming without proper synchronization.
package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	salutation := "hello"
	wg.Add(1)
	go func() {
		defer wg.Done()
		salutation = "welcome" // <1>
	}()
	wg.Wait()
	fmt.Println(salutation)
}

// 1. Here we see that the salutation variable is being modified inside the goroutine. This is an example of a closure.
//    The salutation variable is captured by the goroutine and is shared between the main goroutine and the goroutine.
// The goroutine are execute in the same address space, so they share the same memory.
