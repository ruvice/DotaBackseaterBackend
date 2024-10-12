package main

import (
	"context"
	"fmt"

	"github.com/ruvice/dotabackseaterbackend/application"
)

func main() {
	app := application.New()
	err := app.Start(context.TODO())
	if err != nil {
		fmt.Println("failed to start app", err)
	}

}
