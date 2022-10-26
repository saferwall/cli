// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/joho/godotenv"
	"github.com/saferwall/saferwall-cli/internal/util"
	"github.com/saferwall/saferwall-cli/internal/webapi"
	"github.com/spf13/cobra"
)

const (
	// DefaultAuthUsername represents the key for reading the username
	// from env variables.
	DefaultAuthUsername = "SAFERWALL_AUTH_USERNAME"
	// DefaultAuthPassword represents the key for reading password
	// from env variables.
	DefaultAuthPassword = "SAFERWALL_AUTH_PASSWORD"
)

// Used for flags.
var filePath string
var forceRescanFlag bool
var asyncScanFlag bool

func init() {
	scanCmd.Flags().StringVarP(&filePath, "path", "p", "",
		"File name or path to scan (required)")
	scanCmd.Flags().BoolVarP(&forceRescanFlag, "force", "f", false,
		"Force rescan the file if it exists (default=false)")
	scanCmd.Flags().BoolVarP(&asyncScanFlag, "async", "a", false,
		"Scan files in parallel (default=false)")
	scanCmd.MarkFlagRequired("path")

}

func loadEnv() (username, password string) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	username = os.Getenv(DefaultAuthUsername)
	password = os.Getenv(DefaultAuthPassword)
	if len(username) == 0 || len(password) == 0 {
		log.Fatal("username or password env variables are not set")
	}
	return
}

// scanFile scans an individual file or a directory.
func scanFile(filePath, token string, async, forceRescan bool) error {

	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
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
				data, err := ioutil.ReadFile(filename)
				if err != nil {
					log.Fatalf("failed to read file: %v", filename)
				}
				sha256 := util.GetSha256(data)

				// Check if we the file exists in the DB.
				exists, err := webapi.FileExists(sha256)
				if err != nil {
					log.Fatalf("failed to check existance of file: %v", filename)
				}

				// Upload the file to be scanned, this will automatically
				// triger a scan request.
				if !exists {
					_, err = webapi.Upload(filename, token)
					if err != nil {
						log.Fatalf("failed to upload file: %v", filename)
					}
				} else {
					// Force rescan the file
					if forceRescan {
						err = webapi.Rescan(sha256, token)
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

	// Sequencially scan the files.
	for _, filename := range fileList {
		// Get sha256
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatalf("failed to read file: %v", filename)
		}
		sha256 := util.GetSha256(data)

		log.Printf("processing %s", sha256)

		// Check if we the file exists in the DB.
		exists, err := webapi.FileExists(sha256)
		if err != nil {
			log.Fatalf("failed to check existance of file: %v", filename)
		}

		// Upload the file to be scanned, this will automatically
		// triger a scan request.
		if !exists {
			body, err := webapi.Upload(filename, token)
			if err != nil {
				log.Fatalf("failed to upload file: %v", filename)
			}
			log.Print(body)
		} else {
			// Force rescan the file
			if forceRescan {
				err = webapi.Rescan(sha256, token)
				if err != nil {
					log.Fatalf("failed to rescan file: %v", filename)
				}
			}
		}

		time.Sleep(2 * time.Second)
	}

	return nil
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Submit a scan request of a file using its hash",
	Long:  `Scans the file`,
	Run: func(cmd *cobra.Command, args []string) {

		// load env variable
		username, password := loadEnv()

		// login to saferwall web service
		token, err := webapi.Login(username, password)
		if err != nil {
			log.Fatalf("failed to login to saferwall web service")
		}

		scanFile(filePath, token, asyncScanFlag, forceRescanFlag)
	},
}
