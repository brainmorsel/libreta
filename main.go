package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/brainmorsel/libreta/internal/app"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()
	if err := app.Run(ctx, os.Stdout, os.Stderr, os.Args, os.Getenv); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
