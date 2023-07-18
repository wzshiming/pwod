package main

import (
	"context"
	"os"

	"istio.io/pkg/log"

	"github.com/wzshiming/pwod/pkg/cmd/pwod"
)

func main() {
	ctx := context.Background()
	err := pwod.RootCmd.ExecuteContext(ctx)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
