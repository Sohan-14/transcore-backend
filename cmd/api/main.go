package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sohan-14/transcore-backend/internal/app"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// After the first signal, restore default signal handling: a second
	// Ctrl-C / SIGTERM then kills the process immediately instead of being
	// swallowed while a slow shutdown is in progress.
	go func() {
		<-ctx.Done()
		stop()
	}()

	if err := app.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		return 1
	}
	return 0
}
