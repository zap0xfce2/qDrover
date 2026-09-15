package main

import (
	"fmt"
	"io"
	"os"
)

// version wird beim Build per ldflags gesetzt (siehe Taskfile.yaml), z. B. "v2609121234".
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return runTUI()
	}
	switch args[0] {
	case "send":
		return runSend(args[1:], os.Stdin)
	case "-h", "--help":
		printUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("unbekannter Befehl %q (siehe qdrover --help)", args[0])
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "qDrover — Board mit Herdr-Dispatch-Sender")
	fmt.Fprintln(w, "\nUsage:\n  qdrover\n  qdrover send [query] [--direction up|down|left|right]")
}
