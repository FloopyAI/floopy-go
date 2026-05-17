// Submit NPS-style feedback for a completed chat request.
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
	ctx := context.Background()

	resp, err := client.OpenAI().Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage("ping")},
	})
	if err != nil {
		log.Fatal(err)
	}

	fb, err := client.Feedback.Submit(ctx, floopy.FeedbackSubmitParams{
		Score:     9,
		Useful:    true,
		SessionID: resp.ID,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("feedback recorded (duplicate=%v session=%s)\n", fb.Duplicate, fb.SessionID)
}
