// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package webapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/saferwall/cli/internal/entity"
)

const testBehaviorID = "94b40295-fa4c-5de6-89e5-97cff1e5ecfa"

func TestGetBehaviorReport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		wantPath := "/v1/behaviors/" + testBehaviorID + "/"
		if r.URL.Path != wantPath {
			t.Errorf("path = %s, want %s", r.URL.Path, wantPath)
		}
		if got := r.URL.Query().Get("fields"); got != "env,capabilities" {
			t.Errorf("fields = %q, want %q", got, "env,capabilities")
		}

		json.NewEncoder(w).Encode(map[string]any{
			"env": map[string]any{"sandbox_version": "1.2.3", "guest_healthy": true},
			"capabilities": []map[string]any{
				{"description": "Creates scheduled task", "severity": "high", "category": "persistence"},
			},
		})
	}))
	defer srv.Close()

	svc := New(srv.URL)
	var doc entity.Behavior
	err := svc.GetBehaviorReport(testBehaviorID, []string{"env", "capabilities"}, &doc)
	if err != nil {
		t.Fatalf("GetBehaviorReport() error = %v", err)
	}
	if doc.Environment.SandboxVersion != "1.2.3" {
		t.Errorf("sandbox_version = %q, want 1.2.3", doc.Environment.SandboxVersion)
	}
	if !doc.Environment.GuestHealthy {
		t.Error("guest_healthy = false, want true")
	}
	if len(doc.Capabilities) != 1 || doc.Capabilities[0].Severity != "high" {
		t.Errorf("capabilities = %+v, want one high-severity entry", doc.Capabilities)
	}
}

func TestGetBehaviorReportEmptyFields(t *testing.T) {
	svc := New("http://unused")
	var doc entity.Behavior
	if err := svc.GetBehaviorReport(testBehaviorID, nil, &doc); err == nil {
		t.Fatal("GetBehaviorReport() with empty fields should error")
	}
}

func TestGetBehaviorReportBadField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "field not allowed"})
	}))
	defer srv.Close()

	svc := New(srv.URL)
	var doc entity.Behavior
	err := svc.GetBehaviorReport(testBehaviorID, []string{"bogus"}, &doc)
	if err == nil {
		t.Fatal("GetBehaviorReport() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "field not allowed") {
		t.Errorf("error = %q, want it to contain the API message", err)
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error = %q, want it to contain the status code", err)
	}
}

func TestCountSysEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/v1/behaviors/" + testBehaviorID + "/sys-events/"
		if r.URL.Path != wantPath {
			t.Errorf("path = %s, want %s", r.URL.Path, wantPath)
		}
		q := r.URL.Query()
		if got := q.Get("type"); got != "network" {
			t.Errorf("type = %q, want network", got)
		}
		if got := q.Get("page"); got != "1" {
			t.Errorf("page = %q, want 1", got)
		}
		if got := q.Get("per_page"); got != "1" {
			t.Errorf("per_page = %q, want 1", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"page": 1, "per_page": 1, "page_count": 1234, "total_count": 1234,
			"items": []map[string]any{{"pid": "0x1", "type": "network", "path": "1.2.3.4", "op": "TCP"}},
		})
	}))
	defer srv.Close()

	svc := New(srv.URL)
	count, err := svc.CountSysEvents(testBehaviorID, "network")
	if err != nil {
		t.Fatalf("CountSysEvents() error = %v", err)
	}
	if count != 1234 {
		t.Errorf("count = %d, want 1234", count)
	}
}

func TestCountSysEventsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "resource not found"})
	}))
	defer srv.Close()

	svc := New(srv.URL)
	if _, err := svc.CountSysEvents(testBehaviorID, "file"); err == nil {
		t.Fatal("CountSysEvents() error = nil, want error")
	}
}
