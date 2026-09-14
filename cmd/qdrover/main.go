package main

import (
	"fmt"
	"os"
)

// version wird beim Build per ldflags gesetzt (siehe Taskfile.yaml), z. B. "v2609121234".
var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
