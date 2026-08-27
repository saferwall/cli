// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package webapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/saferwall/cli/internal/entity"
)

const (
	testSHA256 = "275a021bbfb6489e54d471899f7db9d1663fc695ec2fe2a2c4538aabf651fd0f"
	testAPIKey = "test-api-key"
)

func TestScan(t *testing.T) {
	sample := filepath.Join(t.TempDir(), "sample.bin")
	content := []byte("fake malware sample")
	if err := os.WriteFile(sample, content, 0644); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/files/" {
			t.Errorf("path = %s, want /v1/files/", r.URL.Path)
		}
		if got := r.Header.Get("X-Api-Key"); got != testAPIKey {
			t.Errorf("X-Api-Key = %q, want %q", got, testAPIKey)
		}

		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}
		for key, want := range map[string]string{
			"skip_detonation": "false",
			"os":              "windows-10-x64",
			"timeout":         "30",
		} {
			if got := r.FormValue(key); got != want {
				t.Errorf("form field %s = %q, want %q", key, got, want)
			}
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("missing file part: %v", err)
		}
		defer file.Close()
		if header.Filename != "sample.bin" {
			t.Errorf("filename = %q, want sample.bin", header.Filename)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"sha256": testSHA256})
	}))
	defer srv.Close()

	svc := New(srv.URL)
	file, err := svc.Scan(sample, testAPIKey, "windows-10-x64", true, 30)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if file.SHA256 != testSHA256 {
		t.Errorf("Scan() sha256 = %q, want %q", file.SHA256, testSHA256)
	}
}

func TestScanUploadFailure(t *testing.T) {
	sample := filepath.Join(t.TempDir(), "sample.bin")
	if err := os.WriteFile(sample, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message": "quota exceeded"}`, http.StatusTooManyRequests)
	}))
	defer srv.Close()

	svc := New(srv.URL)
	if _, err := svc.Scan(sample, testAPIKey, "windows-10-x64", false, 15); err == nil {
		t.Error("Scan() expected error on HTTP 429, got nil")
	}
}

func TestScanMissingFile(t *testing.T) {
	svc := New("http://unused.invalid")
	if _, err := svc.Scan(filepath.Join(t.TempDir(), "nope"), testAPIKey, "windows-10-x64", false, 15); err == nil {
		t.Error("Scan() expected error for missing local file, got nil")
	}
}

func TestRescan(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if want := "/v1/files/" + testSHA256 + "/rescan"; r.URL.Path != want {
			t.Errorf("path = %s, want %s", r.URL.Path, want)
		}
		if got := r.Header.Get("X-Api-Key"); got != testAPIKey {
			t.Errorf("X-Api-Key = %q, want %q", got, testAPIKey)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := New(srv.URL)
	if err := svc.Rescan(testSHA256, testAPIKey, "windows-7-x64", false, 60,
		true, "US", `C:\\Users\\Public\\tool.exe`,
		`-cmd "cmd /c whoami"`); err != nil {
		t.Fatalf("Rescan() error = %v", err)
	}

	if got := gotBody["skip_detonation"]; got != true {
		t.Errorf("skip_detonation = %v, want true", got)
	}
	if got := gotBody["os"]; got != "windows-7-x64" {
		t.Errorf("os = %v, want windows-7-x64", got)
	}
	if got := gotBody["timeout"]; got != float64(60) {
		t.Errorf("timeout = %v, want 60", got)
	}
	if got := gotBody["network_enabled"]; got != true {
		t.Errorf("network_enabled = %v, want true", got)
	}
	if got := gotBody["country"]; got != "US" {
		t.Errorf("country = %v, want US", got)
	}
	if got := gotBody["dest_path"]; got != `C:\\Users\\Public\\tool.exe` {
		t.Errorf("dest_path = %v, want explicit guest path", got)
	}
	if got := gotBody["args"]; got != `-cmd "cmd /c whoami"` {
		t.Errorf("args = %v, want explicit command line", got)
	}
}

func TestRescanError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "file not found"})
	}))
	defer srv.Close()

	svc := New(srv.URL)
	err := svc.Rescan(testSHA256, testAPIKey, "windows-10-x64", false, 15,
		false, "", "", "")
	if err == nil {
		t.Fatal("Rescan() expected error on HTTP 404, got nil")
	}
	if want := "HTTP 404: file not found"; err.Error() != want {
		t.Errorf("Rescan() error = %q, want %q", err.Error(), want)
	}
}

func TestFileExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("method = %s, want HEAD", r.Method)
		}
		if r.URL.Path == "/v1/files/"+testSHA256 {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	svc := New(srv.URL)

	exists, err := svc.FileExists(testSHA256)
	if err != nil {
		t.Fatalf("FileExists() error = %v", err)
	}
	if !exists {
		t.Error("FileExists() = false for known hash, want true")
	}

	exists, err = svc.FileExists("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("FileExists() error = %v", err)
	}
	if exists {
		t.Error("FileExists() = true for unknown hash, want false")
	}
}

func TestFileExistsServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := New(srv.URL)
	if _, err := svc.FileExists(testSHA256); err == nil {
		t.Error("FileExists() expected error on HTTP 500, got nil")
	}
}

func TestGetFileError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "document does not exist"})
	}))
	defer srv.Close()

	svc := New(srv.URL)
	var file entity.File
	err := svc.GetFile(testSHA256, &file)
	if err == nil {
		t.Fatal("GetFile() expected error on HTTP 404, got nil")
	}
	if want := "HTTP 404: document does not exist"; err.Error() != want {
		t.Errorf("GetFile() error = %q, want %q", err.Error(), want)
	}
}

func TestGetFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/v1/files/" + testSHA256; r.URL.Path != want {
			t.Errorf("path = %s, want %s", r.URL.Path, want)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"sha256":         testSHA256,
			"size":           1024,
			"classification": "Trojan.Generic",
			"file_format":    "pe",
		})
	}))
	defer srv.Close()

	svc := New(srv.URL)
	var file entity.File
	if err := svc.GetFile(testSHA256, &file); err != nil {
		t.Fatalf("GetFile() error = %v", err)
	}
	if file.SHA256 != testSHA256 {
		t.Errorf("sha256 = %q, want %q", file.SHA256, testSHA256)
	}
	if file.Size != 1024 {
		t.Errorf("size = %d, want 1024", file.Size)
	}
	if file.Classification != "Trojan.Generic" {
		t.Errorf("classification = %q, want Trojan.Generic", file.Classification)
	}
}

func TestGetFileStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("fields"); got != "status" {
			t.Errorf("fields query = %q, want status", got)
		}
		json.NewEncoder(w).Encode(map[string]int{"status": 3})
	}))
	defer srv.Close()

	svc := New(srv.URL)
	status, err := svc.GetFileStatus(testSHA256)
	if err != nil {
		t.Fatalf("GetFileStatus() error = %v", err)
	}
	if status != 3 {
		t.Errorf("GetFileStatus() = %d, want 3", status)
	}
}

func TestGetFileStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	svc := New(srv.URL)
	if _, err := svc.GetFileStatus(testSHA256); err == nil {
		t.Error("GetFileStatus() expected error on HTTP 404, got nil")
	}
}

func TestListFiles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Api-Key"); got != testAPIKey {
			t.Errorf("X-Api-Key = %q, want %q", got, testAPIKey)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("page query = %q, want 2", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"page":        2,
			"per_page":    1000,
			"page_count":  5,
			"total_count": 4321,
			"items":       []map[string]string{{"sha256": testSHA256}},
		})
	}))
	defer srv.Close()

	svc := New(srv.URL)
	pages, err := svc.ListFiles(testAPIKey, 2)
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}
	if pages.Page != 2 || pages.PageCount != 5 || pages.TotalCount != 4321 {
		t.Errorf("ListFiles() = %+v, want page=2 page_count=5 total_count=4321", pages)
	}
}

func TestListFilesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid api key"})
	}))
	defer srv.Close()

	svc := New(srv.URL)
	_, err := svc.ListFiles(testAPIKey, 1)
	if err == nil {
		t.Fatal("ListFiles() expected error on HTTP 401, got nil")
	}
	if want := "HTTP 401: invalid api key"; err.Error() != want {
		t.Errorf("ListFiles() error = %q, want %q", err.Error(), want)
	}
}

func TestSearchFiles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/files/search/" {
			t.Errorf("path = %s, want /v1/files/search/", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if got := body["query"]; got != "class:trojan" {
			t.Errorf("query = %v, want class:trojan", got)
		}
		if got := body["page"]; got != float64(1) {
			t.Errorf("page = %v, want 1", got)
		}
		if got := body["per_page"]; got != float64(20) {
			t.Errorf("per_page = %v, want 20", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"page":        1,
			"per_page":    20,
			"page_count":  1,
			"total_count": 1,
			"items": []map[string]any{{
				"id":    testSHA256,
				"class": "trojan",
			}},
		})
	}))
	defer srv.Close()

	svc := New(srv.URL)
	result, err := svc.SearchFiles("class:trojan", testAPIKey, 1, 20)
	if err != nil {
		t.Fatalf("SearchFiles() error = %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("SearchFiles() returned %d items, want 1", len(result.Items))
	}
	if result.Items[0].ID != testSHA256 {
		t.Errorf("item id = %q, want %q", result.Items[0].ID, testSHA256)
	}
	if result.Items[0].Classification != "trojan" {
		t.Errorf("item class = %q, want trojan", result.Items[0].Classification)
	}
}

func TestSearchFilesErrorWithMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid query"})
	}))
	defer srv.Close()

	svc := New(srv.URL)
	_, err := svc.SearchFiles("bogus", testAPIKey, 1, 20)
	if err == nil {
		t.Fatal("SearchFiles() expected error on HTTP 400, got nil")
	}
	if want := "search failed: HTTP 400: invalid query"; err.Error() != want {
		t.Errorf("SearchFiles() error = %q, want %q", err.Error(), want)
	}
}

func TestSearchFilesErrorWithoutMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := New(srv.URL)
	if _, err := svc.SearchFiles("q", testAPIKey, 1, 20); err == nil {
		t.Error("SearchFiles() expected error on HTTP 500, got nil")
	}
}

func TestDownload(t *testing.T) {
	content := []byte("sample bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/v1/files/" + testSHA256 + "/download"; r.URL.Path != want {
			t.Errorf("path = %s, want %s", r.URL.Path, want)
		}
		if got := r.Header.Get("X-Api-Key"); got != testAPIKey {
			t.Errorf("X-Api-Key = %q, want %q", got, testAPIKey)
		}
		w.Write(content)
	}))
	defer srv.Close()

	svc := New(srv.URL)
	buf, err := svc.Download(testSHA256, testAPIKey)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if buf.String() != string(content) {
		t.Errorf("Download() = %q, want %q", buf.String(), content)
	}
}

func TestDownloadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"message": "download not allowed"})
	}))
	defer srv.Close()

	svc := New(srv.URL)
	buf, err := svc.Download(testSHA256, testAPIKey)
	if err == nil {
		t.Fatal("Download() expected error on HTTP 403, got nil")
	}
	if buf != nil {
		t.Error("Download() returned a buffer alongside an error; the error body must not be saved as a sample")
	}
	if want := "HTTP 403: download not allowed"; err.Error() != want {
		t.Errorf("Download() error = %q, want %q", err.Error(), want)
	}
}

func TestDeleteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	svc := New(srv.URL)
	err := svc.Delete(testSHA256, testAPIKey)
	if err == nil {
		t.Fatal("Delete() expected error on HTTP 401, got nil")
	}
	if want := "HTTP 401"; err.Error() != want {
		t.Errorf("Delete() error = %q, want %q", err.Error(), want)
	}
}

func TestDelete(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if want := "/v1/files/" + testSHA256; r.URL.Path != want {
			t.Errorf("path = %s, want %s", r.URL.Path, want)
		}
	}))
	defer srv.Close()

	svc := New(srv.URL)
	if err := svc.Delete(testSHA256, testAPIKey); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !called {
		t.Error("Delete() never reached the server")
	}
}
