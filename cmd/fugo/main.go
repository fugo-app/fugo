package main

import (
	"log"

	"github.com/spf13/cobra"
)

var Version = "0.0.0"

func main() {
	rootCmd := &cobra.Command{
		Use:     "fugo",
		Short:   "Fugo is log parsing and processing agent",
		Version: Version,
	}

	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(watchCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
