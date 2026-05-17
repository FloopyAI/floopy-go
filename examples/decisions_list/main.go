// List + paginate the decision audit trail.
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

	since := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)

	// One page:
	page, err := client.Decisions.List(context.Background(), floopy.DecisionListParams{From: since, Limit: 20})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("first page: %d decisions (has_more=%v)\n", len(page.Items), page.HasMore)

	// Every decision across all pages:
	var n int
	for d, err := range client.Decisions.Iterate(context.Background(), floopy.DecisionListParams{From: since}) {
		if err != nil {
			log.Fatal(err)
		}
		n++
		_ = d
	}
	fmt.Printf("total decisions in window: %d\n", n)
}
