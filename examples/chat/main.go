// Basic chat completion — a drop-in for the openai-go SDK.
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
	opts := []floopy.ClientOption{
		floopy.WithOptions(floopy.Options{
			Cache:              &floopy.CacheOptions{Enabled: floopy.Ptr(true), BucketMaxSize: floopy.Ptr(3)},
			LLMSecurityEnabled: floopy.Ptr(true),
		}),
	}
	if base := os.Getenv("FLOOPY_BASE_URL"); base != "" {
		opts = append(opts, floopy.WithBaseURL(base))
	}

	client, err := floopy.NewClient(os.Getenv("FLOOPY_API_KEY"), opts...)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.OpenAI().Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a concise assistant."),
			openai.UserMessage("Say hi from Floopy in one sentence."),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.Choices[0].Message.Content)
}
