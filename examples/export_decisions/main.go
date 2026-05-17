// Stream the JSONL decision export and capture the trailer.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/FloopyAI/floopy-go"
)

func main() {
	opts := []floopy.ClientOption{}
	if base := os.Getenv("FLOOPY_BASE_URL"); base != "" {
		opts = append(opts, floopy.WithBaseURL(base))
	}
	client, err := floopy.NewClient(os.Getenv("FLOOPY_API_KEY"), opts...)
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now().UTC()
	params := floopy.ExportDecisionsParams{
		From: now.Add(-7 * 24 * time.Hour).Format(time.RFC3339),
		To:   now.Format(time.RFC3339),
	}

	stream := client.Export.DecisionsWithTrailer(context.Background(), params)
	var n int
	for row, err := range stream.Rows {
		if err != nil {
			log.Fatal(err)
		}
		n++
		_ = row
	}
	fmt.Printf("exported %d rows\n", n)
	if tr := stream.Trailer(); tr != nil {
		fmt.Printf("trailer: emitted=%d truncated=%v\n", tr.RowsEmitted, tr.Truncated)
	}
}
