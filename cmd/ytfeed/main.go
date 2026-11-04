package main

import (
	"context"
	"os"

	"github.com/rktkov/ytfeed/internal/transport/cli"
)

func main() {
	os.Exit(cli.Run(context.Background(), os.Args[1:], os.Getenv, os.Stdout, os.Stderr, ""))
}
