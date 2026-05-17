package floopy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func decisionWire(rid string) map[string]any {
	return map[string]any{
		"request_id": rid, "session_id": nil, "request_created_at": "2026-05-10T00:00:00Z",
		"provider": nil, "model": nil, "status": "ok", "latency_ms": nil,
		"cost_micro_usd": nil, "cache_enabled": nil, "threat": nil,
		"decision_trace": nil, "confidence": nil, "confidence_reason": nil, "explanation": nil,
	}
}

func experimentWire(eid string) map[string]any {
	return map[string]any{
		"id": eid, "name": "test", "description": nil, "status": "active",
		"variant_a_routing_rule_id": "rule_a", "variant_b_routing_rule_id": "rule_b",
		"split_percentage": 50, "created_at": "2026-05-10T00:00:00Z", "rolled_back_at": nil,
	}
}

// newTestClient returns a client and records the last request seen.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := NewClient("fl_test", WithBaseURL(srv.URL+"/v1"), WithMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func TestFeedbackPostsBody(t *testing.T) {
	var body map[string]any
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		writeJSON(w, 200, map[string]any{"duplicate": false, "session_id": "s1"})
	})
	res, err := c.Feedback.Submit(context.Background(), FeedbackSubmitParams{Score: 9, Useful: true, SessionID: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Duplicate || res.SessionID != "s1" {
		t.Errorf("unexpected response %+v", res)
	}
	if body["score"].(float64) != 9 || body["useful"] != true || body["session_id"] != "s1" {
		t.Errorf("unexpected request body %+v", body)
	}
}

func TestDecisionsGetMapsWire(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		d := decisionWire("req_1")
		d["latency_ms"] = 123
		d["cache_enabled"] = true
		d["confidence"] = 0.9
		writeJSON(w, 200, d)
	})
	d, err := c.Decisions.Get(context.Background(), "req_1")
	if err != nil {
		t.Fatal(err)
	}
	if d.RequestID != "req_1" || d.LatencyMs == nil || *d.LatencyMs != 123 || d.CacheEnabled == nil || !*d.CacheEnabled {
		t.Errorf("unexpected decision %+v", d)
	}
}

func TestDecisionsPaginate(t *testing.T) {
	var calls int
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			writeJSON(w, 200, map[string]any{"items": []any{decisionWire("req_1")}, "next_cursor": "cur_1", "has_more": true})
			return
		}
		writeJSON(w, 200, map[string]any{"items": []any{decisionWire("req_2")}, "next_cursor": nil, "has_more": false})
	})
	var ids []string
	for page, err := range c.Decisions.Pages(context.Background(), DecisionListParams{From: "2026-05-01T00:00:00Z"}) {
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range page.Items {
			ids = append(ids, d.RequestID)
		}
	}
	if strings.Join(ids, ",") != "req_1,req_2" {
		t.Errorf("ids = %v", ids)
	}
}

func TestExperimentsInjectConfirmHeader(t *testing.T) {
	var seen []string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get(headerConfirm))
		writeJSON(w, 200, experimentWire("exp_1"))
	})
	ctx := context.Background()
	if _, err := c.Experiments.Create(ctx, ExperimentCreateParams{Name: "test", VariantARoutingRuleID: "rule_a", VariantBRoutingRuleID: "rule_b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Experiments.Rollback(ctx, "exp_1"); err != nil {
		t.Fatal(err)
	}
	for _, v := range seen {
		if v != ConfirmExperiments {
			t.Errorf("expected confirm header %q, got %q", ConfirmExperiments, v)
		}
	}
}

func TestConstraintsPutSendsNulls(t *testing.T) {
	var method string
	var body map[string]any
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		writeJSON(w, 200, map[string]any{
			"cost_limit_monthly_usd": 100, "token_window_seconds": nil,
			"max_tokens_per_window": nil, "max_requests_per_minute": 60,
		})
	})
	res, err := c.Constraints.Put(context.Background(), OrgConstraints{
		CostLimitMonthlyUSD:  Ptr(100.0),
		MaxRequestsPerMinute: Ptr(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	if method != "PUT" {
		t.Errorf("method = %q", method)
	}
	if _, ok := body["token_window_seconds"]; !ok {
		t.Errorf("full-replace PUT must send all keys, got %+v", body)
	}
	if res.CostLimitMonthlyUSD == nil || *res.CostLimitMonthlyUSD != 100 || res.TokenWindowSeconds != nil {
		t.Errorf("unexpected response %+v", res)
	}
}

func TestExportSkipsTrailer(t *testing.T) {
	rows := []map[string]any{
		{"request_id": "req_1", "session_id": nil, "organization_id": "org_1", "provider": "openai", "model": "gpt-4o", "status": "ok", "latency_ms": 100, "cost_micro_usd": 1000, "cache_enabled": false, "threat": nil, "created_at": "2026-05-10T00:00:00Z"},
		{"request_id": "req_2", "session_id": nil, "organization_id": "org_1", "provider": "openai", "model": "gpt-4o", "status": "ok", "latency_ms": 200, "cost_micro_usd": 2000, "cache_enabled": false, "threat": nil, "created_at": "2026-05-10T00:01:00Z"},
		{"trailer": true, "rows_emitted": 2, "truncated": false, "reason": nil},
	}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		for _, row := range rows {
			b, _ := json.Marshal(row)
			_, _ = w.Write(append(b, '\n'))
		}
	})
	var ids []string
	for row, err := range c.Export.Decisions(context.Background(), ExportDecisionsParams{From: "a", To: "b"}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, row.RequestID)
	}
	if strings.Join(ids, ",") != "req_1,req_2" {
		t.Errorf("ids = %v", ids)
	}
}

func TestExportWithTrailerCapture(t *testing.T) {
	rows := []map[string]any{
		{"request_id": "req_1", "session_id": nil, "organization_id": "org_1", "provider": nil, "model": nil, "status": "ok", "latency_ms": nil, "cost_micro_usd": nil, "cache_enabled": nil, "threat": nil, "created_at": "2026-05-10T00:00:00Z"},
		{"trailer": true, "rows_emitted": 1, "truncated": true, "reason": "deadline"},
	}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		for _, row := range rows {
			b, _ := json.Marshal(row)
			_, _ = w.Write(append(b, '\n'))
		}
	})
	stream := c.Export.DecisionsWithTrailer(context.Background(), ExportDecisionsParams{From: "a", To: "b"})
	var n int
	for _, err := range stream.Rows {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 1 {
		t.Errorf("expected 1 row, got %d", n)
	}
	tr := stream.Trailer()
	if tr == nil || !tr.Truncated || tr.Reason == nil || *tr.Reason != "deadline" {
		t.Errorf("unexpected trailer %+v", tr)
	}
}

func TestRoutingExplainMapsWire(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"would_select":      map[string]string{"provider": "openai", "model": "gpt-4o-mini"},
			"firewall_decision": "allow", "reasoning": nil, "routing_rule_id": "rule_1",
		})
	})
	res, err := c.Routing.Explain(context.Background(), RoutingExplainParams{Model: "gpt-4o"})
	if err != nil {
		t.Fatal(err)
	}
	if res.WouldSelect["model"] != "gpt-4o-mini" || res.FirewallDecision != "allow" {
		t.Errorf("unexpected result %+v", res)
	}
}

func TestEvaluationsCreate(t *testing.T) {
	var body map[string]any
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		writeJSON(w, 201, map[string]any{
			"id": "eval_1", "dataset_id": "ds_1", "model": "gpt-4o", "prompt_id": nil,
			"status": "pending", "config": nil, "created_at": "2026-05-10T00:00:00Z",
			"started_at": nil, "finished_at": nil,
		})
	})
	run, err := c.Evaluations.Create(context.Background(), EvaluationCreateParams{DatasetID: "ds_1", Model: "gpt-4o"})
	if err != nil {
		t.Fatal(err)
	}
	if run.ID != "eval_1" {
		t.Errorf("id = %q", run.ID)
	}
	if body["dataset_id"] != "ds_1" || body["model"] != "gpt-4o" {
		t.Errorf("unexpected body %+v", body)
	}
	if _, ok := body["prompt_id"]; ok {
		t.Errorf("optional prompt_id should be omitted, got %+v", body)
	}
}

func TestSessionsGetEncodesPath(t *testing.T) {
	var path, trace string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.EscapedPath()
		trace = r.Header.Get("x-trace")
		writeJSON(w, 200, map[string]any{
			"session_id": "sess/1",
			"messages":   []any{map[string]any{"role": "user", "content": "hi"}},
			"turn_count": 1,
			"turns":      []any{map[string]any{"request_id": "r1", "created_at": "2026-05-17T10:00:00Z", "model": "gpt-4o", "provider": "openai"}},
		})
	})
	s, err := c.Sessions.Get(context.Background(), "sess/1", WithRequestHeader("x-trace", "abc"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, "sess%2F1") {
		t.Errorf("session id not percent-encoded in path: %q", path)
	}
	if trace != "abc" {
		t.Errorf("per-call header not sent: %q", trace)
	}
	if s.SessionID != "sess/1" || s.TurnCount != 1 || len(s.Turns) != 1 || s.Turns[0].RequestID != "r1" {
		t.Errorf("unexpected session %+v", s)
	}
}
