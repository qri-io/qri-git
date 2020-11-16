package main

import (
	"context"
	"os"

	"github.com/qri-io/qri-git/qrigit"
)

func main() {
	if len(os.Args) < 3 {
		panic(os.Args)
	}

	ctx := context.Background()

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	gi, err := qrigit.NewGitImporter(ctx)
	if err != nil {
		panic(err)
	}
	_, err = gi.ImportGitFile(os.Args[1], wd, os.Args[2])
	if err != nil {
		panic(err)
	}
}
