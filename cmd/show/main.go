package main

import (
	"context"
	"log"

	"github.com/aaronland/go-geotagged-show"
)

func main() {

	ctx := context.Background()
	err := show.Run(ctx)

	if err != nil {
		log.Fatal(err)
	}
}
