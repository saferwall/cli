// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/saferwall/saferwall-cli/internal/util"
	"github.com/saferwall/saferwall-cli/internal/webapi"
	"github.com/spf13/cobra"
)

// Used for flags.
var filePath string
var forceRescanFlag bool
var asyncScanFlag bool
var skipDetonationFlag bool

func init() {
	scanCmd.Flags().StringVarP(&filePath, "path", "p", "",
		"File name or path to scan (required)")
	scanCmd.Flags().BoolVarP(&forceRescanFlag, "force", "f", false,
		"Force rescan the file if it exists (default=false)")
	scanCmd.Flags().BoolVarP(&asyncScanFlag, "async", "a", false,
		"Scan files in parallel (default=false)")
	scanCmd.Flags().BoolVarP(&skipDetonationFlag, "skipDetonation", "d", false,
		"Skip detonation (default=false)")
	scanCmd.MarkFlagRequired("path")

}

// scanFile scans an individual file or a directory.
func scanFile(filePath, token string, async, forceRescan bool) error {

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

	if async {

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
				exists, err := webapi.FileExists(sha256)
				if err != nil {
					log.Fatalf("failed to check existence of file: %v", filename)
				}

				// Upload the file to be scanned, this will automatically trigger a scan request.
				if !exists {
					_, err = webapi.Upload(filename, token)
					if err != nil {
						log.Fatalf("failed to upload file: %v", filename)
					}
				} else {
					// Force rescan the file
					if forceRescan {
						err = webapi.Rescan(sha256, token, skipDetonationFlag)
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
		// Get sha256
		data, err := os.ReadFile(filename)
		if err != nil {
			log.Fatalf("failed to read file: %v", filename)
		}
		sha256 := util.GetSha256(data)

		log.Printf("processing %s", sha256)

		// Check if we the file exists in the DB.
		exists, err := webapi.FileExists(sha256)
		if err != nil {
			log.Fatalf("failed to check existence of file: %s, error: %v", filename, err)
		}

		// Upload the file to be scanned, this will automatically
		// trigger a scan request.
		if !exists {
			body, err := webapi.Upload(filename, token)
			if err != nil {
				log.Fatalf("failed to upload file: %s, error: %v", filename, err)
			}
			log.Print(body)
			time.Sleep(15 * time.Second)
		} else {
			// Force re-scan the file
			if forceRescan {
				err = webapi.Rescan(sha256, token, skipDetonationFlag)
				if err != nil {
					log.Fatalf("failed to re-scan file: %v", filename)
				}
			}
		}

	}

	return nil
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Submit a scan request of a file using its hash",
	Long:  `Scans the file`,
	Run: func(cmd *cobra.Command, args []string) {

		// login to saferwall web service
		token, err := webapi.Login(cfg.Credentials.Username, cfg.Credentials.Password)
		if err != nil {
			log.Fatalf("failed to login to saferwall web service")
		}

		scanFile(filePath, token, asyncScanFlag, forceRescanFlag)
	},
}
