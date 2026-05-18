package floopy

import (
	"context"
	"strconv"
)

// FileObject is a file stored on the upstream provider. v1 targets the
// "batch" purpose (JSONL input + output files). Fields mirror the OpenAI
// file shape; nullable fields are pointers.
type FileObject struct {
	ID        string `json:"id"`
	Object    string `json:"object,omitempty"`
	Bytes     *int64 `json:"bytes,omitempty"`
	CreatedAt *int64 `json:"created_at,omitempty"`
	Filename  string `json:"filename,omitempty"`
	Purpose   string `json:"purpose,omitempty"`
	Status    string `json:"status,omitempty"`
	Deleted   *bool  `json:"deleted,omitempty"`
}

// FileList is the response of FilesService.List.
type FileList struct {
	Object string       `json:"object,omitempty"`
	Data   []FileObject `json:"data"`
}

// FileUploadParams are the arguments for FilesService.Upload.
type FileUploadParams struct {
	// File is the raw file content (forwarded verbatim as multipart).
	File []byte
	// Filename used in the multipart part. Defaults to "file".
	Filename string
	// Purpose of the upload. Use "batch" for batch input files.
	Purpose string
}

// FileListParams filters FilesService.List.
type FileListParams struct {
	Purpose string
	Limit   int
	After   string
}

func (p FileListParams) query() map[string]string {
	q := map[string]string{}
	if p.Purpose != "" {
		q["purpose"] = p.Purpose
	}
	if p.Limit != 0 {
		q["limit"] = strconv.Itoa(p.Limit)
	}
	if p.After != "" {
		q["after"] = p.After
	}
	return q
}

// FilesService manages files for the Batch API. Select the upstream with
// floopy.WithProvider (the floopy-provider header).
type FilesService struct{ t *transport }

// Upload sends a file as multipart/form-data.
func (s *FilesService) Upload(ctx context.Context, params FileUploadParams, opts ...RequestOption) (*FileObject, error) {
	filename := params.Filename
	if filename == "" {
		filename = "file"
	}
	var out FileObject
	_, err := s.t.doMultipart(ctx, "POST", endpointFiles,
		map[string]string{"purpose": params.Purpose},
		"file", filename, params.File, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns files, optionally filtered by purpose.
func (s *FilesService) List(ctx context.Context, params FileListParams, opts ...RequestOption) (*FileList, error) {
	var out FileList
	_, err := s.t.do(ctx, "GET", endpointFiles, nil, params.query(), &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Get retrieves a single file's metadata.
func (s *FilesService) Get(ctx context.Context, fileID string, opts ...RequestOption) (*FileObject, error) {
	var out FileObject
	_, err := s.t.do(ctx, "GET", fileByID(fileID), nil, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Content downloads raw file content (e.g. a batch output/error JSONL).
func (s *FilesService) Content(ctx context.Context, fileID string, opts ...RequestOption) ([]byte, error) {
	raw, _, err := s.t.doBytes(ctx, "GET", fileContent(fileID), nil, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// Delete removes a file.
func (s *FilesService) Delete(ctx context.Context, fileID string, opts ...RequestOption) (*FileObject, error) {
	var out FileObject
	_, err := s.t.do(ctx, "DELETE", fileByID(fileID), nil, nil, &out, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}
	return &out, nil
}
