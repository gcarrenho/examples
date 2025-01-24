// Package main
//
// This program demonstrates a common pitfall when using closures inside goroutines:
// capturing loop variables. The `salutation` variable, defined in the loop,
// is shared among all the spawned goroutines. As a result, each goroutine
// prints the final value of `salutation` ("good day") rather than the value
// it held during the iteration in which the goroutine was created.
//
// Key Concepts:
//   - Closures capturing loop variables: A loop variable is shared across all iterations,
//     and its value may change before the goroutine executes.
//   - Synchronization with sync.WaitGroup to ensure all goroutines complete before the program exits.
//
// Takeaway:
// To avoid this issue, you can pass the loop variable as an argument to the goroutine function,
// ensuring each goroutine gets its own copy of the variable. This is a common pattern
// for safely working with goroutines and closures in Go.
package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	for _, salutation := range []string{"hello", "greetings", "good day"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println(salutation) // <1>
		}()
	}
	wg.Wait()
}

// 1. Here we print the salutation variable. The salutation variable is shared among all the goroutines, so the output will be the same for all the goroutines.
//	The output will be the last value of the salutation variable, which is "good day".
