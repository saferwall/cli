// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/saferwall/cli/internal/util"
	"github.com/saferwall/cli/internal/webapi"
	"github.com/spf13/cobra"
)

var (
	rescanFilePath string
	fileHash       string
)

func init() {
	reScanCmd.Flags().StringVarP(&rescanFilePath, "path", "p", "",
		"File name or path containing list of SHA256 to scan")
	reScanCmd.Flags().StringVarP(&fileHash, "hash", "s", "",
		"SHA256 of the file to rescan")
	reScanCmd.Flags().IntVar(&parallelFlag, "parallel", 1,
		"Number of files to rescan in parallel")
	reScanCmd.Flags().BoolVarP(&enableDetonationFlag, "enableDetonation", "d", false,
		"Skip detonation")
	reScanCmd.Flags().IntVarP(&timeoutFlag, "timeout", "t", 15,
		"Detonation duration in seconds")
	reScanCmd.Flags().StringVarP(&osFlag, "os", "o", "win-10",
		"Preferred OS for detonation, choice(win-7 | win-10)")
}

// reScanFile re-scans a list of SHA256.
func reScanFile(web webapi.Service, shaList []string, token string) error {
	sem := make(chan struct{}, parallelFlag)
	var wg sync.WaitGroup

	for _, sha256 := range shaList {
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer func() { <-sem; wg.Done() }()
			log.Printf("rescanning %s", sha256)
			err := web.Rescan(sha256, token, osFlag, enableDetonationFlag, timeoutFlag)
			if err != nil {
				log.Printf("failed to rescan file: %v", sha256)
			}
			time.Sleep(2 * time.Second)
		}()
	}
	wg.Wait()
	return nil
}

var reScanCmd = &cobra.Command{
	Use:   "rescan",
	Short: "Rescan an exiting file using its hash",
	Long:  `Rescans the file`,
	Run: func(cmd *cobra.Command, args []string) {

		// Login to saferwall web service
		webSvc := webapi.New(cfg.Credentials.URL)
		token, err := webSvc.Login(cfg.Credentials.Username, cfg.Credentials.Password)
		if err != nil {
			log.Fatalf("failed to login to saferwall web service")
		}

		// Read the txt file containing the list of hashes to rescan.
		var sha256List []string
		if rescanFilePath != "" {
			data, err := util.ReadAll(rescanFilePath)
			if err != nil {
				log.Fatalf("failed to read txt file")
			}

			sha256List = strings.Split(string(data), "\n")
		} else {
			sha256List = append(sha256List, fileHash)
		}

		reScanFile(webSvc, sha256List, token)
	},
}
