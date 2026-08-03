// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package entity

// Behavior status lifecycle values.
const (
	BehaviorStatusQueued     = "queued"
	BehaviorStatusProcessing = "processing"
	BehaviorStatusCompleted  = "completed"
	BehaviorStatusPartial    = "partial"
	BehaviorStatusFailed     = "failed"
)

// BehaviorReportSummary is the bounded per-run summary embedded in a
// file document under `behavior_reports`.
type BehaviorReportSummary struct {
	ID               string              `json:"id"`
	SchemaVersion    int                 `json:"schema_version"`
	Status           string              `json:"status"`
	Revision         int                 `json:"revision"`
	QueuedAt         int64               `json:"queued_at"`
	StartedAt        int64               `json:"started_at,omitempty"`
	FinishedAt       int64               `json:"finished_at,omitempty"`
	ScanConfig       BehaviorScanSummary `json:"scan_config"`
	AttemptCount     int                 `json:"attempt_count,omitempty"`
	Evidence         BehaviorEvidence    `json:"evidence"`
	ArtifactCount    int                 `json:"artifact_count,omitempty"`
	ScreenshotsCount int                 `json:"screenshots_count,omitempty"`
	Failure          *BehaviorFailure    `json:"failure,omitempty"`
}

// BehaviorScanSummary is the sandbox configuration subset embedded in a
// file document.
type BehaviorScanSummary struct {
	OS        string `json:"os,omitempty"`
	ProfileID string `json:"profile_id,omitempty"`
	Timeout   int    `json:"timeout"`
	Country   string `json:"country,omitempty"`
	DestPath  string `json:"dest_path,omitempty"`
}

// BehaviorEvidence stores the detection-evidence rank vector of a run.
type BehaviorEvidence struct {
	MalwareYARA       int `json:"malware_yara"`
	HighBehavior      int `json:"high_behavior"`
	HighOther         int `json:"high_other"`
	Medium            int `json:"medium"`
	TotalRules        int `json:"total_rules"`
	DetectedArtifacts int `json:"detected_artifacts"`
}

// BehaviorFailure describes the terminal failure of a run, when present.
type BehaviorFailure struct {
	Class      string `json:"class"`
	Stage      string `json:"stage,omitempty"`
	RetryClass string `json:"retry_class,omitempty"`
	ExitCode   int    `json:"exit_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

// Behavior is the subset of a behavior document the CLI displays. Fetched
// from /v1/behaviors/{id}/ with an explicit `fields` projection.
type Behavior struct {
	Environment  BehaviorEnvironment `json:"env"`
	Capabilities []Capability        `json:"capabilities,omitempty"`
	ProcessTree  []Process           `json:"proc_tree,omitempty"`
}

// BehaviorEnvironment describes the sandbox used for a run.
type BehaviorEnvironment struct {
	SandboxVersion string `json:"sandbox_version,omitempty"`
	RunID          string `json:"run_id,omitempty"`
	ProfileID      string `json:"profile_id,omitempty"`
	Complete       bool   `json:"complete,omitempty"`
	GuestHealthy   bool   `json:"guest_healthy,omitempty"`
}

// Capability is a behavioral technique detected during execution.
type Capability struct {
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Module      string `json:"module"`
	RuleID      string `json:"rule_id"`
	ProcessID   string `json:"pid"`
}

// Process is one node of the detonation process tree.
type Process struct {
	ImagePath   string `json:"path"`
	PID         string `json:"pid"`
	ParentPID   string `json:"parent_pid"`
	ParentLink  string `json:"parent_link"`
	ProcessName string `json:"proc_name"`
	FileType    string `json:"file_type"`
	Detection   string `json:"detection"`
}
