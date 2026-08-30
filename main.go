// Command wobble is a workload simulator: it runs for a variable amount of time
// (bounded by a hard maximum), burns a variable amount of CPU, and exits with a
// success or failure code chosen by a configurable probability.
package main

import (
	"fmt"
	"os"

	"github.com/wobble/internal/config"
	"github.com/wobble/internal/exitcode"
	"github.com/wobble/internal/run"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	res, err := config.Parse(os.Args[1:], os.Getenv, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wobble: %v\n", err)
		os.Exit(exitcode.Usage)
	}
	switch {
	case res.Usage != "":
		fmt.Fprint(os.Stdout, res.Usage)
		os.Exit(exitcode.Success)
	case res.Version != "":
		fmt.Fprint(os.Stdout, res.Version)
		os.Exit(exitcode.Success)
	}
	os.Exit(run.Execute(res.Config, os.Stderr))
}
