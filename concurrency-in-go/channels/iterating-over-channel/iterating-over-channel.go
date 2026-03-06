package main

import (
	"fmt"
)

func main() {
	intStream := make(chan int)
	go func() {
		defer close(intStream) // <1>
		for i := 1; i <= 5; i++ {
			intStream <- i
		}
	}()

	for integer := range intStream { // <2>
		fmt.Printf("%v ", integer)
	}
}

// 1. Here we close the intStream channel when the goroutine is done.
// 2. Here we iterate over the intStream channel until it is closed.
