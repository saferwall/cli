// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"strings"
	"testing"

	"github.com/saferwall/cli/internal/entity"
)

func summary(status string, finishedAt int64) entity.BehaviorReportSummary {
	return entity.BehaviorReportSummary{Status: status, FinishedAt: finishedAt}
}

func TestSelectBehaviorID(t *testing.T) {
	reports := map[string]entity.BehaviorReportSummary{
		"aaa": summary(entity.BehaviorStatusCompleted, 100),
		"bbb": summary(entity.BehaviorStatusCompleted, 200),
		"ccc": summary(entity.BehaviorStatusFailed, 300),
	}

	tests := []struct {
		name      string
		reports   map[string]entity.BehaviorReportSummary
		defaultID string
		requested string
		want      string
		wantErr   bool
	}{
		{
			name:    "requested wins regardless of status",
			reports: reports, defaultID: "aaa", requested: "ccc",
			want: "ccc",
		},
		{
			name:    "requested not found is an error",
			reports: reports, defaultID: "aaa", requested: "zzz",
			wantErr: true,
		},
		{
			name:    "completed default wins over newer completed run",
			reports: reports, defaultID: "aaa",
			want: "aaa",
		},
		{
			name:    "non-completed default falls back to newest completed",
			reports: reports, defaultID: "ccc",
			want: "bbb",
		},
		{
			name:    "missing default falls back to newest completed",
			reports: reports, defaultID: "",
			want: "bbb",
		},
		{
			name: "started_at breaks ties when finished_at is zero",
			reports: map[string]entity.BehaviorReportSummary{
				"old": {Status: entity.BehaviorStatusCompleted, StartedAt: 10},
				"new": {Status: entity.BehaviorStatusCompleted, StartedAt: 20},
			},
			want: "new",
		},
		{
			name: "no completed run selects nothing",
			reports: map[string]entity.BehaviorReportSummary{
				"aaa": summary(entity.BehaviorStatusFailed, 100),
				"bbb": summary(entity.BehaviorStatusQueued, 0),
			},
			defaultID: "aaa",
			want:      "",
		},
		{
			name:    "empty map selects nothing",
			reports: map[string]entity.BehaviorReportSummary{},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectBehaviorID(tt.reports, tt.defaultID, tt.requested)
			if (err != nil) != tt.wantErr {
				t.Fatalf("selectBehaviorID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("selectBehaviorID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func proc(pid, ppid, name string) entity.Process {
	return entity.Process{PID: pid, ParentPID: ppid, ProcessName: name}
}

func treePIDs(roots []*procNode) []string {
	var pids []string
	var walk func(n *procNode)
	walk = func(n *procNode) {
		pids = append(pids, n.proc.PID)
		for _, c := range n.children {
			walk(c)
		}
	}
	for _, r := range roots {
		walk(r)
	}
	return pids
}

func TestBuildProcTree(t *testing.T) {
	t.Run("nests children under parents", func(t *testing.T) {
		roots := buildProcTree([]entity.Process{
			proc("0x2", "0x1", "child"),
			proc("0x1", "", "root"),
			proc("0x3", "0x2", "grandchild"),
		})
		if len(roots) != 1 {
			t.Fatalf("roots = %d, want 1", len(roots))
		}
		got := treePIDs(roots)
		want := []string{"0x1", "0x2", "0x3"}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("tree order = %v, want %v", got, want)
		}
	})

	t.Run("orphan parent becomes root, children sorted by PID", func(t *testing.T) {
		roots := buildProcTree([]entity.Process{
			proc("0x5", "0x9", "orphan"),
			proc("0x1", "", "root"),
			proc("0x3", "0x1", "b"),
			proc("0x2", "0x1", "a"),
		})
		if len(roots) != 2 {
			t.Fatalf("roots = %d, want 2", len(roots))
		}
		got := treePIDs(roots)
		want := []string{"0x1", "0x2", "0x3", "0x5"}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("tree order = %v, want %v", got, want)
		}
	})

	t.Run("self-parent and cycles terminate and keep all processes", func(t *testing.T) {
		roots := buildProcTree([]entity.Process{
			proc("0x1", "0x1", "self"),
			proc("0x2", "0x3", "cycleA"),
			proc("0x3", "0x2", "cycleB"),
		})
		got := treePIDs(roots)
		if len(got) != 3 {
			t.Errorf("rendered %d processes, want all 3 (got %v)", len(got), got)
		}
	})

	t.Run("duplicate PIDs keep first occurrence", func(t *testing.T) {
		roots := buildProcTree([]entity.Process{
			proc("0x1", "", "first"),
			proc("0x1", "", "second"),
		})
		if len(roots) != 1 || roots[0].proc.ProcessName != "first" {
			t.Errorf("roots = %+v, want single node named first", roots)
		}
	})
}

func TestRenderProcTree(t *testing.T) {
	roots := buildProcTree([]entity.Process{
		proc("0x1", "", "root.exe"),
		proc("0x2", "0x1", "child.exe"),
		proc("0x3", "0x2", "grandchild.exe"),
	})
	lines := renderProcTree(roots)
	if len(lines) != 3 {
		t.Fatalf("lines = %d, want 3", len(lines))
	}
	wantPrefixes := []string{"└─ root.exe", "  └─ child.exe", "    └─ grandchild.exe"}
	for i, want := range wantPrefixes {
		if !strings.HasPrefix(lines[i], want) {
			t.Errorf("line %d = %q, want prefix %q", i, lines[i], want)
		}
	}
}

func TestRenderProcTreeDetection(t *testing.T) {
	p := proc("0x1", "", "evil.exe")
	p.Detection = "Emotet"
	lines := renderProcTree(buildProcTree([]entity.Process{p}))
	if len(lines) != 1 || !strings.Contains(lines[0], "Emotet") {
		t.Errorf("lines = %v, want detection name rendered", lines)
	}

	clean := proc("0x2", "", "ok.exe")
	clean.Detection = "clean"
	lines = renderProcTree(buildProcTree([]entity.Process{clean}))
	if len(lines) != 1 || strings.Contains(lines[0], "clean") {
		t.Errorf("lines = %v, want clean detection omitted", lines)
	}
}
