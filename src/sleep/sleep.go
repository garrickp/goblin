package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	// Handle the case where no argument was passed
	if len(os.Args) < 2 {
		return
	}

	duration, err := strconv.ParseFloat(os.Args[1], 64)

	// Nicely error if an invalid value was passed
	if err != nil {
		fmt.Fprintf(os.Stderr, "sleep: invalid time interval '%s'\n", os.Args[1])
		os.Exit(1)
	}

	time.Sleep(time.Duration(duration) * time.Second)

}
