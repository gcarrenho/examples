// Package main
//
// This program demonstrates the use of `sync.Mutex` to synchronize access
// to a shared variable in concurrent programming. It features two operations:
// incrementing and decrementing a shared `count` variable. The use of a mutex
// ensures that these operations are performed safely, even when multiple
// goroutines access the variable simultaneously.
//
// Key Concepts:
// - `sync.Mutex` to enforce exclusive access to a critical section.
// - `defer` for clean and predictable unlocking of the mutex after the critical section.
// - `sync.WaitGroup` for coordinating the completion of all goroutines before program termination.
// - Managing concurrent operations with explicit locking to prevent race conditions.
//
// This example highlights how to use Go's low-level synchronization primitives
// to safely manage shared resources across multiple goroutines. It provides
// a clear demonstration of mutex locking and unlocking to maintain data integrity.
package main

import (
	"fmt"
	"sync"
)

func main() {
	var count int
	var lock sync.Mutex

	increment := func() {
		lock.Lock()         // <1>
		defer lock.Unlock() // <2>
		count++
		fmt.Printf("Incrementing: %d\n", count)
	}

	decrement := func() {
		lock.Lock()         // <1>
		defer lock.Unlock() // <2>
		count--
		fmt.Printf("Decrementing: %d\n", count)
	}

	// Increment
	var arithmetic sync.WaitGroup
	for i := 0; i <= 5; i++ {
		arithmetic.Add(1)
		go func() {
			defer arithmetic.Done()
			increment()
		}()
	}

	// Decrement
	for i := 0; i <= 5; i++ {
		arithmetic.Add(1)
		go func() {
			defer arithmetic.Done()
			decrement()
		}()
	}

	arithmetic.Wait()
	fmt.Println("Arithmetic complete.")
}

// 1. Here we request exclusive access to the shared `count` variable(critical section). Garantee by the mutex lock.
// 2. Here we release the lock after the critical section is complete.
