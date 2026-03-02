// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/saferwall/cli/internal/entity"
	"github.com/saferwall/cli/internal/util"
	"github.com/saferwall/cli/internal/webapi"
	"github.com/spf13/cobra"
)

const (
	statusQueued    = 1
	statusScanning  = 2
	statusCompleted = 3

	pollInterval = 5 * time.Second
	pollTimeout  = 5 * time.Minute
)

// Used for flags.
var forceRescanFlag bool
var asyncScanFlag bool
var enableDetonationFlag bool
var timeoutFlag int
var osFlag string

func init() {
	scanCmd.Flags().BoolVarP(&forceRescanFlag, "force", "f", false,
		"Force rescan the file if it exists")
	scanCmd.Flags().BoolVarP(&asyncScanFlag, "async", "a", false,
		"Scan files in parallel")
	scanCmd.Flags().BoolVarP(&enableDetonationFlag, "enableDetonation", "d", false,
		"Skip detonation")
	scanCmd.Flags().IntVarP(&timeoutFlag, "timeout", "t", 15,
		"Detonation duration in seconds")
	scanCmd.Flags().StringVarP(&osFlag, "os", "o", "win-10",
		"Preferred OS for detonation, choice(win-7 | win-10)")
}

type scanSummary struct {
	SHA256        string     `json:"sha256"`
	FileFormat    string     `json:"file_format"`
	FileExtension string     `json:"file_extension"`
	MultiAV       *avSummary `json:"multiav,omitempty"`
}

type avSummary struct {
	Positives    int `json:"positives"`
	EnginesCount int `json:"engines_count"`
}

func buildScanSummary(file entity.File) scanSummary {
	s := scanSummary{
		SHA256:        file.SHA256,
		FileFormat:    file.Format,
		FileExtension: file.Extension,
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

// waitForScanCompletion polls the API until the scan completes or times out,
// then pretty-prints the full file report as JSON.
func waitForScanCompletion(web webapi.Service, sha256 string) error {
	deadline := time.Now().Add(pollTimeout)
	for {
		status, err := web.GetFileStatus(sha256)
		if err != nil {
			return fmt.Errorf("failed to poll scan status: %w", err)
		}

		if status == statusCompleted {
			var file entity.File
			if err := web.GetFile(sha256, &file); err != nil {
				return fmt.Errorf("failed to get file report: %w", err)
			}

			summary := buildScanSummary(file)
			pretty, err := json.MarshalIndent(summary, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal scan summary: %w", err)
			}
			fmt.Println(string(pretty))
			return nil
		}

		if time.Now().After(deadline) {
			log.Printf("scan timed out for %s, check later", sha256)
			return nil
		}

		log.Printf("waiting for scan to complete (status=%d)...", status)
		time.Sleep(pollInterval)
	}
}

// scanFile scans an individual file or a directory.
func scanFile(web webapi.Service, filePath, token string) error {

	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		log.Printf("file path [%s] does not exists", filePath)
		return err
	}

	// Walk over directory.
	fileList := []string{}
	filepath.Walk(filePath, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			fileList = append(fileList, path)
		}
		return nil
	})

	if asyncScanFlag {

		// Create a worker pool
		maxWorkers := runtime.GOMAXPROCS(0)
		wp := workerpool.New(maxWorkers)

		// Upload files
		for _, filename := range fileList {
			filename := filename
			wp.Submit(func() {

				// Get sha256
				data, err := os.ReadFile(filename)
				if err != nil {
					log.Fatalf("failed to read file: %v", filename)
				}
				sha256 := util.GetSha256(data)

				// Check if we the file exists in the DB.
				exists, err := web.FileExists(sha256)
				if err != nil {
					log.Fatalf("failed to check existence of file: %v", filename)
				}

				// Upload the file to be scanned, this will automatically trigger a scan request.
				if !exists {
					_, err = web.Scan(filename, token, osFlag, enableDetonationFlag, timeoutFlag)
					if err != nil {
						log.Fatalf("failed to upload file: %v", filename)
					}
				} else {
					// Force rescan the file
					if forceRescanFlag {
						err = web.Rescan(sha256, token, osFlag, enableDetonationFlag, timeoutFlag)
						if err != nil {
							log.Fatalf("failed to rescan file: %v", filename)
						}
					}
				}

				time.Sleep(2 * time.Second)
			})
		}
		wp.StopWait()
		return nil
	}

	// Sequentially scan the files.
	for _, filename := range fileList {
		data, err := os.ReadFile(filename)
		if err != nil {
			log.Fatalf("failed to read file: %v", filename)
		}
		sha256 := util.GetSha256(data)

		log.Printf("processing %s", sha256)

		// Check if the file exists in the DB.
		exists, err := web.FileExists(sha256)
		if err != nil {
			log.Fatalf("failed to check existence of file: %s, error: %v", filename, err)
		}

		// trigger a scan request.
		if !exists {
			_, err := web.Scan(filename, token, osFlag, enableDetonationFlag, timeoutFlag)
			if err != nil {
				log.Fatalf("failed to upload file: %s, error: %v", filename, err)
			}
		} else if forceRescanFlag {
			// Force re-scan the file
			err = web.Rescan(sha256, token, osFlag, enableDetonationFlag, timeoutFlag)
			if err != nil {
				log.Fatalf("failed to re-scan file: %v", filename)
			}
		}

		if err := waitForScanCompletion(web, sha256); err != nil {
			log.Fatalf("error waiting for scan: %v", err)
		}

	}

	return nil
}

var scanCmd = &cobra.Command{
	Use:   "scan <path>",
	Short: "Submit a scan request of a file using its hash",
	Long:  `Scans the file`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		// login to saferwall web service
		webSvc := webapi.New(cfg.Credentials.URL)
		token, err := webSvc.Login(cfg.Credentials.Username, cfg.Credentials.Password)
		if err != nil {
			log.Fatalf("failed to login to saferwall web service")
		}

		scanFile(webSvc, args[0], token)
	},
}
