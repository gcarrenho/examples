// Package main
//
// This program demonstrates the use of `sync.Cond` to implement an event
// subscription system in Go. It simulates a scenario where multiple
// goroutines are waiting for a "button click" event, and once the event
// is broadcasted, all subscribed goroutines are triggered to perform
// their respective actions.
//
// Key Concepts:
//   - `sync.Cond` as an event signaling mechanism for goroutines.
//   - Use of `Broadcast` to notify all goroutines waiting on a condition.
//   - Goroutine synchronization using `sync.WaitGroup` to ensure proper
//     coordination and orderly execution.
//   - Encapsulation of event-driven behavior within a custom type (`Button`)
//     and the `subscribe` function.
//
// This example illustrates how to build a simple publish-subscribe
// mechanism in Go, leveraging `sync.Cond` to coordinate goroutines
// efficiently. It highlights the power of Go's concurrency primitives
// to manage event-based workflows in concurrent programs.
package main

import (
	"fmt"
	"sync"
)

func main() {
	type Button struct { // <1>
		Clicked *sync.Cond
	}
	button := Button{Clicked: sync.NewCond(&sync.Mutex{})}

	subscribe := func(c *sync.Cond, fn func()) { // <2>
		var goroutineRunning sync.WaitGroup
		goroutineRunning.Add(1)
		go func() {
			goroutineRunning.Done()
			c.L.Lock()
			defer c.L.Unlock()
			c.Wait()
			fn()
		}()
		goroutineRunning.Wait()
	}

	var clickRegistered sync.WaitGroup // <3>
	clickRegistered.Add(3)
	subscribe(button.Clicked, func() { // <4>
		fmt.Println("Maximizing window.")
		clickRegistered.Done()
	})
	subscribe(button.Clicked, func() { // <5>
		fmt.Println("Displaying annoying dialogue box!")
		clickRegistered.Done()
	})
	subscribe(button.Clicked, func() { // <6>
		fmt.Println("Mouse clicked.")
		clickRegistered.Done()
	})

	button.Clicked.Broadcast() // <7>

	clickRegistered.Wait()
}

// 1. Here we define a Button type with a `Clicked` field of type `*sync.Cond`.
// 2. Here we define a `subscribe` function that takes a `*sync.Cond` and a function as arguments. The function starts a goroutine that waits for the condition to be signaled and then calls the function.
// 3. Here we define a `sync.WaitGroup` to wait for the goroutines to complete.
// 4. Here we subscribe to the button click event and print a message when the event is triggered.
// 5. Here we subscribe to the button click event and print a message when the event is triggered.
// 6. Here we subscribe to the button click event and print a message when the event is triggered.
// 7. Here we broadcast the button click event to all subscribers.
