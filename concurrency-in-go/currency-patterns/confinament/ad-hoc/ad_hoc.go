package main

// We can  that the data slice of integer is available from both the loopData function and the loop over the handleData channel
// However, by ocnvention we are only accessing the data slice from the loopData function. But as the code is touched by many people
// and deadline loom, mistakes might be made, and the confinament might break down and cause issue.
// A static analysis tool might help to catch such issues, but static analysis on a Go codebase suggest a level of maturity that not many teams achive. So for this we prefer lexical confinament. it is better to use a more robust solution.
import (
	"fmt"
)

func main() {
	data := make([]int, 4)

	loopData := func(handleData chan<- int) {
		defer close(handleData)
		for i := range data {
			handleData <- data[i]
		}
	}

	handleData := make(chan int)
	go loopData(handleData)

	for num := range handleData {
		fmt.Println(num)
	}
}
