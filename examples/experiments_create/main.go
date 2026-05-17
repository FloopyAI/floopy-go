// Create + roll back a routing experiment. The X-Floopy-Confirm header is
// injected by the SDK automatically.
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

	exp, err := client.Experiments.Create(ctx, floopy.ExperimentCreateParams{
		Name:                  "cost-vs-quality",
		VariantARoutingRuleID: os.Getenv("FLOOPY_RULE_A"),
		VariantBRoutingRuleID: os.Getenv("FLOOPY_RULE_B"),
		SplitPercentage:       floopy.Ptr(50),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("created experiment %s (%s)\n", exp.ID, exp.Status)

	results, err := client.Experiments.Results(ctx, exp.ID)
	if err == nil {
		fmt.Printf("variant A sample size: %d\n", results.VariantA.SampleSize)
	}

	if _, err := client.Experiments.Rollback(ctx, exp.ID); err != nil {
		log.Fatal(err)
	}
	fmt.Println("rolled back")
}
