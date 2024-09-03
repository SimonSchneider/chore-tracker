package main

import (
	"context"
	"fmt"
	"github.com/SimonSchneider/go-testing/chore"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

func main() {
	if err := chore.Run(context.Background(), os.Args, os.Stdin, os.Stdout, os.Stderr, os.Getenv, os.Getwd); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
