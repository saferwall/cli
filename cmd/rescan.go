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

func init() {
	reScanCmd.Flags().StringVarP(&filePath, "path", "p", "",
		"File name or path containing list of SHA256 to scan (required)")
	reScanCmd.Flags().BoolVarP(&asyncScanFlag, "async", "a", false,
		"Scan files in parallel (default=false)")
	reScanCmd.MarkFlagRequired("path")

}

// reScanFile re-scans a list of SHA256.
func reScanFile(shaList []string, token string, async bool) error {

	if async {
		// Create a worker pool
		maxWorkers := runtime.GOMAXPROCS(0)
		wp := workerpool.New(maxWorkers)

		for _, sha256 := range shaList {
			wp.Submit(func() {
				log.Printf("rescanning %s", sha256)
				err := webapi.Rescan(sha256, token)
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

		log.Printf("rescanning %s", sha256)
		err := webapi.Rescan(sha256, token)
		if err != nil {
			log.Fatalf("failed to rescan file: %v", sha256)
		}
		time.Sleep(15 * time.Second)

	}

	return nil
}

var reScanCmd = &cobra.Command{
	Use:   "rescan",
	Short: "Rescan an exiting file using its hash",
	Long:  `Rescans the file`,
	Run: func(cmd *cobra.Command, args []string) {

		// load env variable
		username, password := loadEnv()

		// login to saferwall web service
		token, err := webapi.Login(username, password)
		if err != nil {
			log.Fatalf("failed to login to saferwall web service")
		}

		// read the txt file containing the list of hashes to
		// rescan.
		data, err := util.ReadAll(filePath)
		if err != nil {
			log.Fatalf("failed to read txt file")
		}

		sha256List := strings.Split(string(data), "\n")

		reScanFile(sha256List, token, asyncScanFlag)
	},
}
