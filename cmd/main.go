package main

import (
	"context"
	"fmt"
	"github.com/SimonSchneider/go-testing/internal/chore"
	"os"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func main() {
	if err := chore.Run(context.Background(), os.Args, os.Stdin, os.Stdout, os.Stderr, os.Getenv, os.Getwd); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
