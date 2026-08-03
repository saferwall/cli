// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saferwall/cli/internal/entity"
	"github.com/saferwall/cli/internal/util"
	"github.com/saferwall/cli/internal/webapi"
	"github.com/spf13/cobra"
)

const (
	statusCompleted = 3

	pollInterval = 5 * time.Second
)

// Used for flags.
var forceRescanFlag bool
var parallelFlag int
var enableDetonationFlag bool
var timeoutFlag int
var osFlag string

func init() {
	scanCmd.Flags().BoolVarP(&forceRescanFlag, "force", "f", false,
		"Force rescan the file if it exists")
	scanCmd.Flags().IntVarP(&parallelFlag, "parallel", "p", 1,
		"Number of files to scan in parallel")
	scanCmd.Flags().BoolVarP(&enableDetonationFlag, "enableDetonation", "d", false,
		"Enable sandbox detonation (skipped by default)")
	scanCmd.Flags().IntVarP(&timeoutFlag, "timeout", "t", 15,
		"Detonation duration in seconds")
	scanCmd.Flags().StringVarP(&osFlag, "os", "o", "windows-10-x64",
		"Preferred OS for detonation, choice(windows-7-x64 | windows-10-x64 | windows-11-x64)")
}

type scanSummary struct {
	SHA256             string     `json:"sha256"`
	Size               int64      `json:"size"`
	Classification     string     `json:"classification"`
	FileFormat         string     `json:"file_format"`
	FileExtension      string     `json:"file_extension"`
	Encrypted          bool       `json:"encrypted,omitempty"`
	DecryptionSuccess  *bool      `json:"decryption_success,omitempty"`
	SuccessfulPassword string     `json:"successful_password,omitempty"`
	AttemptedPasswords []string   `json:"attempted_passwords,omitempty"`
	MultiAV            *avSummary `json:"multiav,omitempty"`
}

type avSummary struct {
	Positives    int `json:"positives"`
	EnginesCount int `json:"engines_count"`
}

func buildScanSummary(file entity.File) scanSummary {
	s := scanSummary{
		SHA256:             file.SHA256,
		Size:               file.Size,
		Classification:     file.Classification,
		FileFormat:         file.Format,
		FileExtension:      file.Extension,
		Encrypted:          file.Encrypted,
		DecryptionSuccess:  file.DecryptionSuccess,
		SuccessfulPassword: file.SuccessfulPassword,
		AttemptedPasswords: file.AttemptedPasswords,
	}

	if lastScan, ok := file.MultiAV["last_scan"].(map[string]any); ok {
		if stats, ok := lastScan["stats"].(map[string]any); ok {
			av := &avSummary{}
			if v, ok := stats["positives"].(float64); ok {
				av.Positives = int(v)
			}
			if v, ok := stats["engines_count"].(float64); ok {
				av.EnginesCount = int(v)
			}
			s.MultiAV = av
		}
	}

	return s
}

// scanFile scans an individual file or a directory.
func scanFile(web webapi.Service, filePath, token string) error {
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("cannot access %s: %w", filePath, err)
	}

	// Walk over the file or directory.
	fileList, err := util.WalkAllFilesInDir(filePath)
	if err != nil {
		return fmt.Errorf("failed walking %s: %w", filePath, err)
	}
	if len(fileList) == 0 {
		return fmt.Errorf("no files found in %s", filePath)
	}

	// Launch TUI scan with the configured parallelism.
	model := newScanModel(fileList, web, token, parallelFlag)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

var scanCmd = &cobra.Command{
	Use:   "scan <path>",
	Short: "Submit a scan request of a file using its hash",
	Long:  `Scans the file`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		webSvc := webapi.New(cfg.Credentials.URL)
		return scanFile(webSvc, args[0], cfg.Credentials.APIKey)
	},
}
