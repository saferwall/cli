// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/saferwall/cli/internal/util"
	"github.com/saferwall/cli/internal/webapi"
	"github.com/spf13/cobra"
)

var sha256Re = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

func init() {
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
	Use:   "rescan <sha256|file>",
	Short: "Rescan an existing file using its hash",
	Long:  `Rescans one or more files. Pass a SHA256 hash to rescan a single file, or a path to a text file with one hash per line to rescan in batch.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		// Login to saferwall web service
		webSvc := webapi.New(cfg.Credentials.URL)
		token, err := webSvc.Login(cfg.Credentials.Username, cfg.Credentials.Password)
		if err != nil {
			log.Fatalf("failed to login to saferwall web service")
		}

		arg := args[0]

		var sha256List []string
		if sha256Re.MatchString(arg) {
			sha256List = append(sha256List, arg)
		} else {
			data, err := util.ReadAll(arg)
			if err != nil {
				log.Fatalf("failed to read file: %s", arg)
			}
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					sha256List = append(sha256List, line)
				}
			}
		}

		reScanFile(webSvc, sha256List, token)
	},
}
