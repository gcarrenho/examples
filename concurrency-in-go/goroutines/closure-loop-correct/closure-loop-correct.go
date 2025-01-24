// Package main
//
// This program demonstrates the correct way to handle loop variables in goroutines by
// passing the loop variable as an argument to the goroutine function. Unlike the common
// pitfall where the loop variable is shared among goroutines, this approach ensures each
// goroutine gets its own copy of the variable.
//
// Key Concepts:
//   - Passing loop variables as arguments: By passing the `salutation` variable as a parameter
//     to the anonymous function, each goroutine operates on a unique copy of the variable.
//   - Synchronization with sync.WaitGroup: Used to wait for all goroutines to complete
//     before exiting the program.
//
// Takeaway:
// This approach prevents unexpected behavior when using goroutines in a loop,
// ensuring that each goroutine works with the intended value of the loop variable.
package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	for _, salutation := range []string{"hello", "greetings", "good day"} {
		wg.Add(1)
		go func(salutation string) { // <1>
			defer wg.Done()
			fmt.Println(salutation)
		}(salutation) // <2>
	}
	wg.Wait()
}

// 1. Here we pass the salutation variable as an argument to the goroutine function,
// 	ensuring each goroutine gets its own copy of the variable.
// 2. Here we pass the salutation variable to the goroutine function.
