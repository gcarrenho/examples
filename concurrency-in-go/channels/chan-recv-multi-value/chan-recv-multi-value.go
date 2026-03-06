package main

import (
	"fmt"
)

func main() {
	stringStream := make(chan string)
	go func() {
		stringStream <- "Hello channels!"
	}()
	salutation, ok := <-stringStream // <1>
	fmt.Printf("(%v): %v", ok, salutation)
}

// 1. Here we receive the string from the stringStream channel. The ok value will be true if the channel is open and false if the channel is closed.
