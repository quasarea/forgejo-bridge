package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/quasarea/forgejo-bridge/internal/cli"
	"github.com/quasarea/forgejo-bridge/internal/mcpserver"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	args := os.Args[1:]
	if len(args) >= 2 && args[0] == "mcp" && args[1] == "stdio" {
		if err := mcpserver.RunStdio(ctx, args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(8)
		}
		return
	}

	runner := cli.Runner{Stdout: os.Stdout, Stderr: os.Stderr}
	os.Exit(runner.Run(ctx, args))
}
