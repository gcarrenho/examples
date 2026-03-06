package main

import (
	"fmt"
)

func main() {
	intStream := make(chan int)
	close(intStream)
	integer, ok := <-intStream // <1>
	fmt.Printf("(%v): %v", ok, integer)
}

// 1. Here we try to read from the closed intStream channel. The integer value will be the zero value of the channel's type, and the ok value will be false.
