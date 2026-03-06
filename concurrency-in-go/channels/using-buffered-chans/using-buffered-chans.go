package main

import (
	"bytes"
	"fmt"
	"os"
)

func main() {
	var stdoutBuff bytes.Buffer         // <1>
	defer stdoutBuff.WriteTo(os.Stdout) // <2>

	intStream := make(chan int, 4) // <3>
	go func() {
		defer close(intStream)
		defer fmt.Fprintln(&stdoutBuff, "Producer Done.")
		for i := 0; i < 5; i++ {
			fmt.Fprintf(&stdoutBuff, "Sending: %d\n", i)
			intStream <- i
		}
	}()

	for integer := range intStream {
		fmt.Fprintf(&stdoutBuff, "Received %v.\n", integer)
	}
}

// 1. Here we create an in-memory buffer to help mitigate the nondeterministic nature of
// the output. It does not give us any guarantees, but its a little faster than writing to stdout directly.
// 2. We defer the writing of the buffer to stdout. This ensures that the buffer is written to stdout before process exits.
// 3. We create a buffered channel of integers with capacity of four.
