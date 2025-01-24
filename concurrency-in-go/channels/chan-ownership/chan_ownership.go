package main

import (
	"fmt"
)

func main() {
	chanOwner := func() <-chan int {
		resultStream := make(chan int, 5) // <1>
		go func() {                       // <2>
			defer close(resultStream) // <3>
			for i := 0; i <= 5; i++ {
				resultStream <- i
			}
		}()
		return resultStream // <4>
	}

	resultStream := chanOwner()
	for result := range resultStream { // <5>
		fmt.Printf("Received: %d\n", result)
	}
	fmt.Println("Done receiving!")
}

// 1. Here we instantiante a buffered channel. Since we know we will produce six results, we create a buffered channel
// of five so that the goroutinen can complete as quickly as posible.
// 2. Here we start an anonymous goroutine that performs writes on resultStream channel. Notice that we have inverted how we create
// goroutines. It is now encapsulated within the surronunding function.
// 3. Hre we ensure resultStream is closed when the goroutine completes. As the channel owner, this is our responsibility.
// 4. Here we return the channel. Since the return value is declarated as a read-only channel, resultStream will implicity
// be converted to a read-only channel for consumer.
// 5. Here we loop over the channel. As a consumer, we are only concerned with blocking and closed channels.
