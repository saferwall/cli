// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saferwall/cli/internal/util"
	"github.com/saferwall/cli/internal/webapi"
	"github.com/spf13/cobra"
)

var outputFlag string
var extractFlag bool

func init() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	downloadCmd.Flags().StringVarP(&outputFlag, "output", "o", filepath.Dir(ex),
		"Destination directory where to save samples. (default=current dir)")
	downloadCmd.Flags().IntVarP(&parallelFlag, "parallel", "p", 4,
		"Number of files to download in parallel")
	downloadCmd.Flags().BoolVarP(&extractFlag, "extract", "x", false,
		"Extract samples from zip (password: infected)")
}

var downloadCmd = &cobra.Command{
	Use:   "download <sha256|file.txt>",
	Short: "Download a sample (and its artifacts)",
	Long:  `Download a binary sample given a SHA256 hash, or a batch of samples from a text file containing one hash per line.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		arg := args[0]

		// Login to saferwall web service.
		webSvc := webapi.New(cfg.Credentials.URL)
		token, err := webSvc.Login(cfg.Credentials.Username, cfg.Credentials.Password)
		if err != nil {
			log.Fatalf("failed to authenticate: %v", err)
		}

		hashes := collectHashes(arg)
		if len(hashes) == 0 {
			log.Fatalf("no valid SHA256 hashes found in %q", arg)
		}

		downloadFiles(webSvc, token, hashes)
	},
}

// collectHashes returns a list of SHA256 hashes from the argument.
// If arg is a SHA256 hash, it returns a single-element slice.
// Otherwise it treats arg as a file path and reads hashes from it.
func collectHashes(arg string) []string {
	if sha256Re.MatchString(arg) {
		return []string{arg}
	}

	data, err := util.ReadAll(arg)
	if err != nil {
		log.Fatalf("failed to read SHA256 hashes from file: %s", arg)
	}

	var hashes []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if sha256Re.MatchString(line) {
			hashes = append(hashes, line)
		}
	}
	return hashes
}

func downloadFiles(web webapi.Service, token string, hashes []string) {
	model := newDownloadModel(hashes, web, token, outputFlag, parallelFlag, extractFlag)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
