// Routing dry-run (Pro plan): see which provider/model the gateway would
// pick — and the firewall verdict — without calling a provider.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/FloopyAI/floopy-go"
	"github.com/openai/openai-go/v3"
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

	res, err := client.Routing.Explain(context.Background(), floopy.RoutingExplainParams{
		Model: "gpt-4o",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Summarise the war and peace in one line."),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("firewall: %s\n", res.FirewallDecision)
	if res.WouldSelect != nil {
		fmt.Printf("would route to: %s / %s\n", res.WouldSelect["provider"], res.WouldSelect["model"])
	} else {
		fmt.Println("blocked by firewall")
	}
}
