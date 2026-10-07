package floopy

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNativeResponsesAndStreaming(t *testing.T) {
	output := `{"id":"resp_test","object":"response","created_at":1,"model":"gpt-6-luna","status":"completed","output":[{"id":"msg_1","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Olá","annotations":[]}]}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" || r.Method != "POST" {
			t.Errorf("unexpected endpoint %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer fl_test" || r.Header.Get("floopy-llm-security-enabled") != "true" {
			t.Error("gateway headers missing")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["input"] != "hi" {
			t.Errorf("native input lost: %v", body)
		}
		if body["stream"] == true {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"Olá\",\"sequence_number\":0,\"item_id\":\"msg_1\",\"output_index\":0,\"content_index\":0}\n\nevent: response.completed\ndata: {\"type\":\"response.completed\",\"sequence_number\":1,\"response\":%s}\n\n", output)
		} else {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, output)
		}
	}))
	defer srv.Close()
	c, err := NewClient("fl_test", WithBaseURL(srv.URL+"/v1"), WithMaxRetries(0), WithOptions(Options{LLMSecurityEnabled: Ptr(true)}))
	if err != nil {
		t.Fatal(err)
	}
	params := responses.ResponseNewParams{Model: "gpt-6-luna", Input: responses.ResponseNewParamsInputUnion{OfString: openai.String("hi")}}
	response, err := c.Responses().New(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if response.OutputText() != "Olá" {
		t.Fatalf("output lost: %s", response.OutputText())
	}
	stream := c.Responses().NewStreaming(context.Background(), params)
	defer stream.Close()
	var kinds []string
	for stream.Next() {
		kinds = append(kinds, stream.Current().Type)
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if len(kinds) != 2 || kinds[0] != "response.output_text.delta" || kinds[1] != "response.completed" {
		t.Fatalf("unexpected events: %v", kinds)
	}
}
