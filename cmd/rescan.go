// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/saferwall/saferwall-cli/internal/util"
	"github.com/saferwall/saferwall-cli/internal/webapi"
	"github.com/spf13/cobra"
)

var (
	fileHash string
)

func init() {
	reScanCmd.Flags().StringVarP(&filePath, "path", "p", "",
		"File name or path containing list of SHA256 to scan")
	reScanCmd.Flags().StringVarP(&fileHash, "hash", "s", "",
		"SHA256 of the file to rescan")
	reScanCmd.Flags().BoolVarP(&asyncScanFlag, "async", "a", false,
		"Scan files in parallel")
	reScanCmd.Flags().BoolVarP(&skipDetonationFlag, "skipDetonation", "d", false,
		"Skip detonation")
	reScanCmd.Flags().IntVarP(&timeoutFlag, "timeout", "t", 15,
		"Detonation duration in seconds")
	reScanCmd.Flags().StringVarP(&osFlag, "os", "o", "win-10",
		"Preferred OS for detonation, choice(win-7 | win-10)")
}

// reScanFile re-scans a list of SHA256.
func reScanFile(shaList []string, token string) error {

	if asyncScanFlag {
		// Create a worker pool
		maxWorkers := runtime.GOMAXPROCS(0)
		wp := workerpool.New(maxWorkers)

		for _, sha256 := range shaList {
			wp.Submit(func() {
				log.Printf("rescanning %s", sha256)
				err := webapi.Rescan(sha256, token, osFlag, skipDetonationFlag, timeoutFlag)
				if err != nil {
					log.Fatalf("failed to rescan file: %v", sha256)
				}

				time.Sleep(2 * time.Second)
			})
		}
		wp.StopWait()
		return nil
	}

	// Sequentially scan the files.
	for _, sha256 := range shaList {

		log.Printf("re-scanning %s", sha256)
		err := webapi.Rescan(sha256, token, osFlag, skipDetonationFlag, timeoutFlag)
		if err != nil {
			log.Fatalf("failed to rescan file: %v", sha256)
		}

		if len(shaList) > 1 {
			time.Sleep(10 * time.Second)
		}

	}

	return nil
}

var reScanCmd = &cobra.Command{
	Use:   "rescan",
	Short: "Rescan an exiting file using its hash",
	Long:  `Rescans the file`,
	Run: func(cmd *cobra.Command, args []string) {

		// Login to saferwall web service
		token, err := webapi.Login(cfg.Credentials.URL, cfg.Credentials.Username, cfg.Credentials.Password)
		if err != nil {
			log.Fatalf("failed to login to saferwall web service")
		}

		// Read the txt file containing the list of hashes to rescan.
		var sha256List []string
		if filePath != "" {
			data, err := util.ReadAll(filePath)
			if err != nil {
				log.Fatalf("failed to read txt file")
			}

			sha256List = strings.Split(string(data), "\n")
		} else {
			sha256List = append(sha256List, fileHash)
		}

		reScanFile(sha256List, token)
	},
}
