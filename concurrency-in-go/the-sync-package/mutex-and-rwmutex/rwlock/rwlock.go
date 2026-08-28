// Package main
//
// This program demonstrates the performance comparison between `sync.Mutex` and `sync.RWMutex`
// under different levels of concurrency. It uses a producer and multiple observers to simulate
// read-write contention on a shared resource and measures the time taken for each scenario.
//
// Key Concepts:
// - `sync.Locker` interface for generic locking and unlocking (accepts both `sync.Mutex` and `sync.RWMutex`).
// - Performance trade-offs between `sync.Mutex` and `sync.RWMutex` in read-dominated scenarios.
// - Measuring execution time of concurrent operations with `time.Since`.
// - Using `tabwriter` to format output for easy comparison.
//
// How It Works:
// - A `producer` function locks and unlocks a shared resource at a slower rate (1-second delay per iteration).
// - An `observer` function locks the resource (using either a read or write lock depending on the test scenario).
// - The `test` function runs a producer alongside multiple observers and records the time taken.
// - Results are printed in a table, showing the impact of using `RWMutex` versus `Mutex` for varying numbers of readers.
//
// This example highlights the importance of choosing the appropriate synchronization primitive
// based on the read-write workload to optimize performance in concurrent applications.
package main

import (
	"fmt"
	"math"
	"os"
	"sync"
	"text/tabwriter"
	"time"
)

func main() {
	producer := func(wg *sync.WaitGroup, l sync.Locker) { // <1>
		defer wg.Done()
		for i := 5; i > 0; i-- {
			l.Lock()
			l.Unlock()
			time.Sleep(1) // <2>
		}
	}

	observer := func(wg *sync.WaitGroup, l sync.Locker) {
		defer wg.Done()
		l.Lock()
		defer l.Unlock()
	}

	test := func(count int, mutex, rwMutex sync.Locker) time.Duration {
		var wg sync.WaitGroup
		wg.Add(count + 1)
		beginTestTime := time.Now()
		go producer(&wg, mutex)
		for i := count; i > 0; i-- {
			go observer(&wg, rwMutex)
		}

		wg.Wait()
		return time.Since(beginTestTime)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 1, 2, ' ', 0)
	defer tw.Flush()

	var m sync.RWMutex
	fmt.Fprintf(tw, "Readers\tRWMutext\tMutex\n")
	for i := 0; i < 20; i++ {
		count := int(math.Pow(2, float64(i)))
		fmt.Fprintf(
			tw,
			"%d\t%v\t%v\n",
			count,
			test(count, &m, m.RLocker()),
			test(count, &m, &m),
		)
	}
}

// 1. The procuder function second parameter is of the type `sync.Locker` which is an interface that defines the `Lock` and `Unlock`
//	 methods. This allows us to pass either a `sync.Mutex` or `sync.RWMutex` to the function.
// 2. Here we make the producer function sleep for 1 second to make it less active than the observer goroutines.
