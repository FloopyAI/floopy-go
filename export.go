package floopy

import (
	"context"
	"encoding/json"
	"iter"
)

// ExportFormat selects the server-side export encoding. The SDK always parses
// the JSONL stream regardless; "csv" is accepted for parity with the gateway.
type ExportFormat = string

const (
	ExportFormatJSONL ExportFormat = "jsonl"
	ExportFormatCSV   ExportFormat = "csv"
)

// ExportedDecisionRow is one row of the decision export.
type ExportedDecisionRow struct {
	RequestID      string  `json:"request_id"`
	SessionID      *string `json:"session_id"`
	OrganizationID string  `json:"organization_id"`
	Provider       *string `json:"provider"`
	Model          *string `json:"model"`
	Status         string  `json:"status"`
	LatencyMs      *int    `json:"latency_ms"`
	CostMicroUSD   *int64  `json:"cost_micro_usd"`
	CacheEnabled   *bool   `json:"cache_enabled"`
	Threat         *string `json:"threat"`
	CreatedAt      string  `json:"created_at"`
}

// ExportTrailer is the terminal record of the JSONL export: it reports how
// many rows were emitted and whether the result was truncated.
type ExportTrailer struct {
	RowsEmitted int     `json:"rows_emitted"`
	Truncated   bool    `json:"truncated"`
	Reason      *string `json:"reason"`
}

// ExportDecisionsParams selects the export window and format.
type ExportDecisionsParams struct {
	From   string       // RFC3339, required
	To     string       // RFC3339, required
	Format ExportFormat // optional
}

func (p ExportDecisionsParams) query() map[string]string {
	q := map[string]string{"from": p.From, "to": p.To}
	if p.Format != "" {
		q["format"] = p.Format
	}
	return q
}

func isTrailer(raw json.RawMessage) bool {
	var probe struct {
		Trailer bool `json:"trailer"`
	}
	_ = json.Unmarshal(raw, &probe)
	return probe.Trailer
}

// ExportService streams the decision log.
type ExportService struct{ t *transport }

// Decisions streams decision rows from the JSONL export. The terminal
// trailer record is skipped — use DecisionsWithTrailer to capture it.
// Iteration stops on the first error (yielded as the second value).
func (s *ExportService) Decisions(ctx context.Context, params ExportDecisionsParams, opts ...RequestOption) iter.Seq2[ExportedDecisionRow, error] {
	return func(yield func(ExportedDecisionRow, error) bool) {
		for raw, err := range s.t.streamLines(ctx, "GET", endpointExportDecisions, params.query(), newRequestConfig(opts)) {
			if err != nil {
				yield(ExportedDecisionRow{}, err)
				return
			}
			if isTrailer(raw) {
				continue
			}
			var row ExportedDecisionRow
			if uerr := json.Unmarshal(raw, &row); uerr != nil {
				yield(ExportedDecisionRow{}, &FloopyError{Message: "failed to decode export row: " + uerr.Error(), cause: uerr})
				return
			}
			if !yield(row, nil) {
				return
			}
		}
	}
}

// DecisionExportStream is an iterable export that also captures the trailer.
type DecisionExportStream struct {
	rows    iter.Seq2[ExportedDecisionRow, error]
	trailer *ExportTrailer
}

// Rows iterates the decision rows. After iteration completes (the stream is
// fully consumed without an early break), Trailer returns the trailer record
// if the gateway sent one.
func (d *DecisionExportStream) Rows(yield func(ExportedDecisionRow, error) bool) {
	d.rows(yield)
}

// Trailer returns the export trailer, or nil if iteration has not completed
// or the gateway sent no trailer.
func (d *DecisionExportStream) Trailer() *ExportTrailer { return d.trailer }

// DecisionsWithTrailer streams decision rows and captures the trailer. Range
// over the returned value's Rows; once iteration finishes, Trailer() holds
// the summary (truncation reason / totals) or nil.
func (s *ExportService) DecisionsWithTrailer(ctx context.Context, params ExportDecisionsParams, opts ...RequestOption) *DecisionExportStream {
	stream := &DecisionExportStream{}
	stream.rows = func(yield func(ExportedDecisionRow, error) bool) {
		for raw, err := range s.t.streamLines(ctx, "GET", endpointExportDecisions, params.query(), newRequestConfig(opts)) {
			if err != nil {
				yield(ExportedDecisionRow{}, err)
				return
			}
			if isTrailer(raw) {
				var tr ExportTrailer
				if json.Unmarshal(raw, &tr) == nil {
					stream.trailer = &tr
				}
				continue
			}
			var row ExportedDecisionRow
			if uerr := json.Unmarshal(raw, &row); uerr != nil {
				yield(ExportedDecisionRow{}, &FloopyError{Message: "failed to decode export row: " + uerr.Error(), cause: uerr})
				return
			}
			if !yield(row, nil) {
				return
			}
		}
	}
	return stream
}
