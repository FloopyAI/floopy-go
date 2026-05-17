// Streaming chat completion. Streaming is delegated to openai-go.
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
	client, err := floopy.NewClient(os.Getenv("FLOOPY_API_KEY"), baseURL()...)
	if err != nil {
		log.Fatal(err)
	}

	stream := client.OpenAI().Chat.Completions.NewStreaming(context.Background(), openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Write a short haiku about gateways."),
		},
	})
	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) > 0 {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}
	if err := stream.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Println()
}

func baseURL() []floopy.ClientOption {
	if base := os.Getenv("FLOOPY_BASE_URL"); base != "" {
		return []floopy.ClientOption{floopy.WithBaseURL(base)}
	}
	return nil
}
