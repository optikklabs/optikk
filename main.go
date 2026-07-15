package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/optikklabs/optikk/cmd"
)

func main() {
	err := cmd.NewRootCmd().Execute()
	if err == nil {
		return
	}
	// A silent exit carries a status, not a failure; the command has already
	// said what it needed to on stdout.
	var silent cmd.SilentExitError
	if errors.As(err, &silent) {
		os.Exit(silent.Code)
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
