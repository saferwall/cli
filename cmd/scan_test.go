// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/saferwall/cli/internal/entity"
)

const testHash = "275a021bbfb6489e54d471899f7db9d1663fc695ec2fe2a2c4538aabf651fd0f"

func TestBuildScanSummary(t *testing.T) {
	success := true
	file := entity.File{
		SHA256:             testHash,
		Size:               2048,
		Classification:     "Ransomware.Wannacry",
		Format:             "pe",
		Extension:          "exe",
		Encrypted:          true,
		DecryptionSuccess:  &success,
		SuccessfulPassword: "infected",
		AttemptedPasswords: []string{"infected", "malware"},
		MultiAV: map[string]any{
			"last_scan": map[string]any{
				"stats": map[string]any{
					"positives":     float64(12),
					"engines_count": float64(14),
				},
			},
		},
	}

	got := buildScanSummary(file)

	if got.SHA256 != testHash {
		t.Errorf("sha256 = %q, want %q", got.SHA256, testHash)
	}
	if got.Size != 2048 {
		t.Errorf("size = %d, want 2048", got.Size)
	}
	if got.Classification != "Ransomware.Wannacry" {
		t.Errorf("classification = %q, want Ransomware.Wannacry", got.Classification)
	}
	if got.FileFormat != "pe" || got.FileExtension != "exe" {
		t.Errorf("format/extension = %q/%q, want pe/exe", got.FileFormat, got.FileExtension)
	}
	if !got.Encrypted || got.DecryptionSuccess == nil || !*got.DecryptionSuccess {
		t.Error("encryption fields not carried over")
	}
	if got.SuccessfulPassword != "infected" {
		t.Errorf("successful_password = %q, want infected", got.SuccessfulPassword)
	}
	if got.MultiAV == nil {
		t.Fatal("multiav summary = nil, want populated")
	}
	if got.MultiAV.Positives != 12 || got.MultiAV.EnginesCount != 14 {
		t.Errorf("multiav = %d/%d, want 12/14", got.MultiAV.Positives, got.MultiAV.EnginesCount)
	}
}

func TestBuildScanSummaryNoMultiAV(t *testing.T) {
	got := buildScanSummary(entity.File{SHA256: testHash})
	if got.MultiAV != nil {
		t.Errorf("multiav = %+v, want nil when last_scan stats are absent", got.MultiAV)
	}
}

func TestSha256Re(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{testHash, true},
		{"275A021BBFB6489E54D471899F7DB9D1663FC695EC2FE2A2C4538AABF651FD0F", true},
		{"not-a-hash", false},
		{testHash[:63], false},
		{testHash + "0", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := sha256Re.MatchString(tt.input); got != tt.want {
			t.Errorf("sha256Re.MatchString(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCollectHashesSingleHash(t *testing.T) {
	got := collectHashes(testHash)
	if !reflect.DeepEqual(got, []string{testHash}) {
		t.Errorf("collectHashes() = %v, want [%s]", got, testHash)
	}
}

func TestCollectHashesFromFile(t *testing.T) {
	other := "0000000000000000000000000000000000000000000000000000000000000000"
	path := filepath.Join(t.TempDir(), "hashes.txt")
	content := testHash + "\n" +
		"  " + other + "  \n" + // surrounding whitespace is trimmed
		"garbage line\n" +
		"\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got := collectHashes(path)
	want := []string{testHash, other}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("collectHashes() = %v, want %v", got, want)
	}
}
