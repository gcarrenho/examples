package main

import (
	"fmt"
)

func main() {
	stringStream := make(chan string)
	go func() {
		stringStream <- "Hello channels!" // <1>
	}()
	fmt.Println(<-stringStream) // <2>
}

// 1. Here we send a string to the stringStream channel.
// 2. Here we receive the string from the stringStream channel.(we are not using a goroutine to receive the string +
//	from the channel, so the main goroutine will block until the string is received from the channel.)
