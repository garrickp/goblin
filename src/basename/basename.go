package main

import (
	"fmt"
	"os"
	"path"
	"strings"
)

func main() {

	argc := len(os.Args) - 1
	dFlag := false
	pathIndex := 1

	if os.Args[1] == "-d" {
		dFlag = true
		pathIndex = 2
	}

	if argc < pathIndex || argc > pathIndex+1 {
		fmt.Fprint(os.Stderr, "Usage: basename [-d] path [suffix]\n")
		os.Exit(1)
	}

	userPath := os.Args[pathIndex]

	if dFlag {
		fmt.Printf("%s\n", path.Dir(userPath))
		return
	}

	suffix := ""
	if len(os.Args) > pathIndex+1 {
		suffix = os.Args[pathIndex+1]
	}

	if len(suffix) > 0 && strings.HasSuffix(userPath, suffix) {
		userPath = strings.TrimSuffix(userPath, suffix)
	}

	fmt.Printf("%s\n", path.Base(userPath))

}
