// Single + batch embeddings, delegated to openai-go.
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

	resp, err := client.OpenAI().Embeddings.New(context.Background(), openai.EmbeddingNewParams{
		Model: openai.EmbeddingModelTextEmbedding3Small,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: []string{"floopy gateway", "go sdk"},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	for i, e := range resp.Data {
		fmt.Printf("embedding %d: %d dimensions\n", i, len(e.Embedding))
	}
}
