// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package webapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/saferwall/cli/internal/entity"
)

// maxErrorBodyLen bounds how much of an error response body is echoed
// back in error messages.
const maxErrorBodyLen = 512

// apiError builds an error from a non-2xx response, preferring the API's
// own `message` field when the body contains one.
func apiError(statusCode int, body []byte) error {
	var jsonBody struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &jsonBody); err == nil && jsonBody.Message != "" {
		return fmt.Errorf("HTTP %d: %s", statusCode, jsonBody.Message)
	}

	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > maxErrorBodyLen {
		trimmed = trimmed[:maxErrorBodyLen]
	}
	if len(trimmed) == 0 {
		return fmt.Errorf("HTTP %d", statusCode)
	}
	return fmt.Errorf("HTTP %d: %s", statusCode, trimmed)
}

// do sends the request and returns the response body. Any non-2xx
// response is converted into an error via apiError.
func (s Service) do(req *http.Request) ([]byte, error) {
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, apiError(resp.StatusCode, body)
	}
	return body, nil
}

func (s Service) newfileUploadRequest(fieldname, filename string, params map[string]string) (*http.Request, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for key, val := range params {
		err := writer.WriteField(key, val)
		if err != nil {
			return nil, err
		}
	}

	part, err := writer.CreateFormFile(fieldname, filepath.Base(filename))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", s.filesURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

// FileExists determines file existence.
func (s Service) FileExists(sha256 string) (bool, error) {
	url := s.filesURL + sha256
	resp, err := s.client.Head(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return false, nil
	case resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices:
		return true, nil
	default:
		return false, apiError(resp.StatusCode, nil)
	}
}

// ListFiles list all the files in DB.
func (s Service) ListFiles(authToken string, page int) (*Pages, error) {
	url := fmt.Sprintf("%s?per_page=%d&page=%d&fields=sha256", s.filesURL, 1000, page)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Api-Key", authToken)

	body, err := s.do(request)
	if err != nil {
		return nil, err
	}

	var pages Pages
	if err := json.Unmarshal(body, &pages); err != nil {
		return nil, err
	}
	return &pages, nil
}

func (s Service) Scan(filepath string, authToken, preferredOS string, enableDetonation bool, timeout int) (*entity.File, error) {
	params := map[string]string{
		"skip_detonation": strconv.FormatBool(!enableDetonation),
		"os":              preferredOS,
		"timeout":         strconv.Itoa(timeout),
	}

	// Create a new file upload request.
	request, err := s.newfileUploadRequest("file", filepath, params)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Api-Key", authToken)

	body, err := s.do(request)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	var file entity.File
	if err := json.Unmarshal(body, &file); err != nil {
		return nil, fmt.Errorf("failed to parse upload response: %w", err)
	}
	return &file, nil
}

func (s Service) Rescan(sha256, authToken, preferredOS string, enableDetonation bool, timeout int) error {
	url := s.filesURL + sha256 + "/rescan"

	requestBody, err := json.Marshal(map[string]any{
		"skip_detonation": !enableDetonation,
		"os":              preferredOS,
		"timeout":         timeout,
	})
	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Api-Key", authToken)

	_, err = s.do(request)
	return err
}

// GetFile retrieves the file report given a sha256.
func (s Service) GetFile(sha256 string, file *entity.File) error {
	url := s.filesURL + sha256

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	body, err := s.do(req)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, file)
}

// GetFileStatus retrieves only the status field of a file.
func (s Service) GetFileStatus(sha256 string) (int, error) {
	url := s.filesURL + sha256 + "?fields=status"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	body, err := s.do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get file status: %w", err)
	}

	var result struct {
		Status int `json:"status"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}
	return result.Status, nil
}

func (s Service) Download(sha256, authToken string) (*bytes.Buffer, error) {
	url := s.filesURL + sha256 + "/download"
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Api-Key", authToken)

	body, err := s.do(request)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(body), nil
}

func (s Service) Delete(sha256, authToken string) error {
	url := s.filesURL + sha256
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("X-Api-Key", authToken)

	_, err = s.do(request)
	return err
}

// SearchItem is the flattened file representation returned by the search endpoint.
type SearchItem struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Format         string         `json:"file_format"`
	Extension      string         `json:"file_extension"`
	Size           int64          `json:"size"`
	FirstSeen      int64          `json:"first_seen"`
	LastScanned    int64          `json:"last_scanned"`
	Classification string         `json:"class"`
	MultiAV        SearchMultiAV  `json:"multiav"`
	Tags           map[string]any `json:"tags"`
}

// SearchMultiAV holds the condensed AV stats returned in search results.
type SearchMultiAV struct {
	Hits  int `json:"hits"`
	Total int `json:"total"`
}

// SearchResult is the paginated response from the search endpoint.
type SearchResult struct {
	Page       int          `json:"page"`
	PerPage    int          `json:"per_page"`
	PageCount  int          `json:"page_count"`
	TotalCount int          `json:"total_count"`
	Items      []SearchItem `json:"items"`
}

// SearchFiles calls POST /v1/files/search/ with the given query expression.
func (s Service) SearchFiles(query, authToken string, page, perPage int) (*SearchResult, error) {
	url := s.filesURL + "search/"
	requestBody, err := json.Marshal(map[string]any{
		"query":    query,
		"page":     page,
		"per_page": perPage,
	})
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Api-Key", authToken)

	body, err := s.do(request)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
