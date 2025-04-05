package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
)

var Version = "0.0.0"

func main() {
	versionFlag := flag.Bool("version", false, "Print version information")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("fugo version %s\n", Version)
		os.Exit(0)
	}

	handler := slog.NewTextHandler(os.Stdout, nil)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
