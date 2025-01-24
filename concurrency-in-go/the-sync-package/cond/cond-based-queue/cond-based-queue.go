// Package main
//
// This program demonstrates the use of `sync.Cond` in Go to coordinate
// and manage access to a shared resource, specifically a queue with
// limited capacity. The example highlights how goroutines can use
// conditional waits and signals to synchronize their actions.
//
// Key Concepts:
// - `sync.Cond` as a tool for managing coordination between goroutines.
// - Use of `sync.Mutex` to protect shared resources and prevent race conditions.
// - Communication between goroutines using `Wait`, `Signal`, and `Broadcast`.
// - Managing resource limits (queue capacity) in a concurrent environment.
//
// This example emphasizes the importance of protecting shared memory
// with mutexes and using condition variables to wait for and signal
// changes in state. It serves as a practical demonstration of how
// to handle concurrency challenges when working with shared resources.
package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	c := sync.NewCond(&sync.Mutex{})    // <1>
	queue := make([]interface{}, 0, 10) // <2>

	removeFromQueue := func(delay time.Duration) {
		time.Sleep(delay)
		c.L.Lock()        // <8>
		queue = queue[1:] // <9>
		fmt.Println("Removed from queue")
		c.L.Unlock() // <10>
		c.Signal()   // <11>
	}

	for i := 0; i < 10; i++ {
		c.L.Lock()            // <3>
		for len(queue) == 2 { // <4>
			c.Wait() // <5>
		}
		fmt.Println("Adding to queue")
		queue = append(queue, struct{}{})
		go removeFromQueue(1 * time.Second) // <6>
		c.L.Unlock()                        // <7>
	}
}

// 1. Heere we create a new Cond type with a new Mutex type.
// 2. Here we create a queue with a capacity of 10.
// 3. Here we lock the mutex to protect the queue.
// 4. Here we check if the queue is full.
// 5. Here we wait until the queue has space.
// 6. Here we add an element to the queue and start a goroutine to remove it after 1 second.
// 7. Here we unlock the mutex to allow other goroutines to access the queue.
// 8. Here we lock the mutex to protect the queue.
// 9. Here we remove the first element from the queue.
// 10. Here we unlock the mutex to allow other goroutines to access the queue.
// 11. Here we signal that an element has been removed from the queue.
