package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/ruvice/dotabackseaterbackend/application"
)

func main() {
	// Only use context.Background in main!!!
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	app := application.New(ctx, application.LoadConfig())
	// Basically says call cancel() at the end
	defer cancel()
	err := app.Start(ctx)
	if err != nil {
		fmt.Println("failed to start app", err)
	}
}
