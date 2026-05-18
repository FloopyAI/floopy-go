package floopy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFilesUploadMultipart(t *testing.T) {
	var ct, provider, purpose, fname string
	var fileBody []byte
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		ct = r.Header.Get("Content-Type")
		provider = r.Header.Get("floopy-provider")
		_ = r.ParseMultipartForm(1 << 20)
		purpose = r.FormValue("purpose")
		f, hdr, _ := r.FormFile("file")
		fname = hdr.Filename
		fileBody, _ = io.ReadAll(f)
		writeJSON(w, 200, map[string]any{"id": "file-1", "object": "file", "purpose": "batch", "status": "ok"})
	})
	res, err := c.Files.Upload(context.Background(),
		FileUploadParams{File: []byte(`{"x":1}` + "\n"), Filename: "in.jsonl", Purpose: "batch"},
		WithProvider("openai"))
	if err != nil {
		t.Fatal(err)
	}
	if res.ID != "file-1" {
		t.Errorf("id = %q", res.ID)
	}
	if !strings.HasPrefix(ct, "multipart/form-data") {
		t.Errorf("content-type = %q", ct)
	}
	if provider != "openai" || purpose != "batch" || fname != "in.jsonl" {
		t.Errorf("provider=%q purpose=%q fname=%q", provider, purpose, fname)
	}
	if string(fileBody) != `{"x":1}`+"\n" {
		t.Errorf("file body = %q", fileBody)
	}
}

func TestFilesUploadDefaultFilename(t *testing.T) {
	var fname string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(1 << 20)
		_, hdr, _ := r.FormFile("file")
		fname = hdr.Filename
		writeJSON(w, 200, map[string]any{"id": "file-2"})
	})
	if _, err := c.Files.Upload(context.Background(),
		FileUploadParams{File: []byte("x"), Purpose: "batch"}); err != nil {
		t.Fatal(err)
	}
	if fname != "file" {
		t.Errorf("default filename = %q", fname)
	}
}

func TestFilesListQueryVariants(t *testing.T) {
	var rawQuery string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		writeJSON(w, 200, map[string]any{"object": "list", "data": []any{map[string]any{"id": "f1"}}})
	})
	page, err := c.Files.List(context.Background(),
		FileListParams{Purpose: "batch", Limit: 10, After: "f0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "f1" {
		t.Errorf("data = %+v", page.Data)
	}
	if !strings.Contains(rawQuery, "purpose=batch") || !strings.Contains(rawQuery, "limit=10") || !strings.Contains(rawQuery, "after=f0") {
		t.Errorf("query = %q", rawQuery)
	}
	if _, err := c.Files.List(context.Background(), FileListParams{}); err != nil {
		t.Fatal(err)
	}
	if rawQuery != "" {
		t.Errorf("expected empty query, got %q", rawQuery)
	}
}

func TestFilesGetContentDelete(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/files/file-1":
			writeJSON(w, 200, map[string]any{"id": "file-1"})
		case r.Method == "GET" && r.URL.Path == "/v1/files/file-1/content":
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"a":1}` + "\n"))
		case r.Method == "DELETE" && r.URL.Path == "/v1/files/file-1":
			writeJSON(w, 200, map[string]any{"id": "file-1", "deleted": true})
		default:
			w.WriteHeader(404)
		}
	})
	ctx := context.Background()
	if got, err := c.Files.Get(ctx, "file-1"); err != nil || got.ID != "file-1" {
		t.Fatalf("get: %+v %v", got, err)
	}
	body, err := c.Files.Content(ctx, "file-1", WithProvider("openai"))
	if err != nil || string(body) != `{"a":1}`+"\n" {
		t.Fatalf("content: %q %v", body, err)
	}
	del, err := c.Files.Delete(ctx, "file-1")
	if err != nil || del.Deleted == nil || !*del.Deleted {
		t.Fatalf("delete: %+v %v", del, err)
	}
}

func TestBatchesCreateListGetCancel(t *testing.T) {
	var lastBody map[string]any
	var rawQuery string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/v1/batches":
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &lastBody)
			writeJSON(w, 200, map[string]any{
				"id": "batch_1", "status": "validating",
				"request_counts": map[string]any{"total": 2, "completed": 0, "failed": 0},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/batches":
			rawQuery = r.URL.RawQuery
			writeJSON(w, 200, map[string]any{"object": "list", "data": []any{map[string]any{"id": "batch_1"}}, "has_more": false})
		case r.Method == "GET" && r.URL.Path == "/v1/batches/batch_1":
			writeJSON(w, 200, map[string]any{"id": "batch_1", "status": "completed"})
		case r.Method == "POST" && r.URL.Path == "/v1/batches/batch_1/cancel":
			writeJSON(w, 200, map[string]any{"id": "batch_1", "status": "cancelling"})
		default:
			w.WriteHeader(404)
		}
	})
	ctx := context.Background()
	b, err := c.Batches.Create(ctx, BatchCreateParams{
		InputFileID: "file-1", Endpoint: "/v1/chat/completions",
		CompletionWindow: "24h", Metadata: map[string]string{"k": "v"},
	}, WithProvider("openai"))
	if err != nil || b.ID != "batch_1" || b.RequestCounts == nil || *b.RequestCounts.Total != 2 {
		t.Fatalf("create: %+v %v", b, err)
	}
	if lastBody["metadata"].(map[string]any)["k"] != "v" {
		t.Errorf("metadata not sent: %+v", lastBody)
	}
	if _, err := c.Batches.Create(ctx, BatchCreateParams{
		InputFileID: "file-1", Endpoint: "/v1/chat/completions", CompletionWindow: "24h",
	}); err != nil {
		t.Fatal(err)
	}
	page, err := c.Batches.List(ctx, BatchListParams{Limit: 5, After: "batch_0"})
	if err != nil || page.HasMore == nil || *page.HasMore || page.Data[0].ID != "batch_1" {
		t.Fatalf("list: %+v %v", page, err)
	}
	if !strings.Contains(rawQuery, "limit=5") || !strings.Contains(rawQuery, "after=batch_0") {
		t.Errorf("query = %q", rawQuery)
	}
	if _, err := c.Batches.List(ctx, BatchListParams{}); err != nil {
		t.Fatal(err)
	}
	if rawQuery != "" {
		t.Errorf("expected empty query, got %q", rawQuery)
	}
	if got, err := c.Batches.Get(ctx, "batch_1"); err != nil || got.Status != "completed" {
		t.Fatalf("get: %+v %v", got, err)
	}
	if got, err := c.Batches.Cancel(ctx, "batch_1", WithProvider("openai")); err != nil || got.Status != "cancelling" {
		t.Fatalf("cancel: %+v %v", got, err)
	}
}

func TestBatchDecodeEdgeCases(t *testing.T) {
	ctx := context.Background()
	// 204 No Content on delete: decodeJSON returns the zero value, no error.
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	if got, err := c.Files.Delete(ctx, "file-1"); err != nil || got.ID != "" {
		t.Fatalf("204 delete: %+v %v", got, err)
	}
	// Empty 200 body: decodeJSON returns the zero value, no error.
	c2 := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})
	if got, err := c2.Batches.Get(ctx, "batch_1"); err != nil || got.ID != "" {
		t.Fatalf("empty body: %+v %v", got, err)
	}
	// Non-JSON 200 body: decodeJSON surfaces a decode error.
	c3 := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("not json"))
	})
	if _, err := c3.Files.Get(ctx, "file-1"); err == nil {
		t.Fatal("expected a decode error on non-JSON body")
	}
}

func TestBatchAndFilesErrorsPropagate(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 400, map[string]any{"error": map[string]any{"message": "bad"}})
	})
	ctx := context.Background()
	if _, err := c.Files.Upload(ctx, FileUploadParams{File: []byte("x"), Purpose: "batch"}); err == nil {
		t.Error("upload should error")
	}
	if _, err := c.Files.List(ctx, FileListParams{}); err == nil {
		t.Error("list should error")
	}
	if _, err := c.Files.Get(ctx, "f1"); err == nil {
		t.Error("get should error")
	}
	if _, err := c.Files.Content(ctx, "f1"); err == nil {
		t.Error("content should error")
	}
	if _, err := c.Files.Delete(ctx, "f1"); err == nil {
		t.Error("delete should error")
	}
	if _, err := c.Batches.Create(ctx, BatchCreateParams{}); err == nil {
		t.Error("create should error")
	}
	if _, err := c.Batches.List(ctx, BatchListParams{}); err == nil {
		t.Error("list should error")
	}
	if _, err := c.Batches.Get(ctx, "b1"); err == nil {
		t.Error("get should error")
	}
	if _, err := c.Batches.Cancel(ctx, "b1"); err == nil {
		t.Error("cancel should error")
	}
}
