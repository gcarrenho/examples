// Package main
//
// This program demonstrates the memory overhead of creating a large number of goroutines in Go.
// It measures the memory consumed before and after launching `numGoroutines` goroutines,
// calculating the average memory cost per goroutine.
//
// Key Concepts:
// 1. **Memory Consumption Measurement**:
//   - The `memConsumed` function uses `runtime.GC()` and `runtime.ReadMemStats` to get
//     the memory statistics of the program before and after the goroutines are created.
//
// 2. **Blocking with a Nil Channel**:
//   - The `noop` function blocks each goroutine by waiting on a receive operation
//     from a nil channel, preventing them from completing and being garbage collected.
//
// 3. **High Concurrency Testing**:
//   - By setting `numGoroutines` to a high value (10,000 in this case), the program
//     simulates a scenario of high concurrency and calculates the average memory used
//     per goroutine.
//
// Takeaway:
//   - Goroutines in Go are lightweight compared to traditional threads. This program
//     illustrates just how lightweight they are by demonstrating the minimal memory
//     overhead associated with their creation.
//
// Note:
//   - Blocking on a nil channel ensures the goroutines remain active and do not
//     terminate prematurely, which would otherwise distort the memory usage measurement.
package main

import (
	"fmt"
	"runtime"
	"sync"
)

func main() {
	memConsumed := func() uint64 {
		runtime.GC()
		var s runtime.MemStats
		runtime.ReadMemStats(&s)
		return s.Sys
	}

	var c <-chan interface{}
	var wg sync.WaitGroup
	noop := func() { wg.Done(); <-c } // <1>

	const numGoroutines = 1e4 // <2>
	wg.Add(numGoroutines)
	before := memConsumed() // <3>
	for i := numGoroutines; i > 0; i-- {
		go noop()
	}
	wg.Wait()
	after := memConsumed() // <4>
	fmt.Printf("%.3fkb", float64(after-before)/numGoroutines/1000)
}

// 1. Here we define a noop function that decrements the WaitGroup and blocks on a receive from the nil channel.
// 2. Here we define the number of goroutines to create.
// 3. Here we record the memory consumed before creating the goroutines.
// 4. Here we record the memory consumed after creating the goroutines.
