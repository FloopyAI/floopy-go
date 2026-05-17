// Read + full-replace the org spend/rate constraints.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

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
	ctx := context.Background()

	current, err := client.Constraints.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("current monthly cap: %v\n", current.CostLimitMonthlyUSD)

	// Put is full-replace: any nil field is cleared server-side.
	updated, err := client.Constraints.Put(ctx, floopy.OrgConstraints{
		CostLimitMonthlyUSD:  floopy.Ptr(100.0),
		MaxRequestsPerMinute: floopy.Ptr(120),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("new monthly cap: %v\n", *updated.CostLimitMonthlyUSD)
}
