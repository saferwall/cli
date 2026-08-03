// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saferwall/cli/internal/util"
	"github.com/saferwall/cli/internal/webapi"
	"github.com/spf13/cobra"
)

var outputFlag string
var extractFlag bool

func init() {
	downloadCmd.Flags().StringVarP(&outputFlag, "output", "o", ".",
		"Destination directory where to save samples")
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
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := args[0]

		webSvc := webapi.New(cfg.Credentials.URL)
		hashes, err := collectHashes(arg)
		if err != nil {
			return err
		}
		if len(hashes) == 0 {
			return fmt.Errorf("no valid SHA256 hashes found in %q", arg)
		}

		return downloadFiles(webSvc, cfg.Credentials.APIKey, hashes)
	},
}

// collectHashes returns a list of SHA256 hashes from the argument.
// If arg is a SHA256 hash, it returns a single-element slice.
// Otherwise it treats arg as a file path and reads hashes from it.
func collectHashes(arg string) ([]string, error) {
	if sha256Re.MatchString(arg) {
		return []string{arg}, nil
	}

	data, err := util.ReadAll(arg)
	if err != nil {
		return nil, fmt.Errorf("failed to read SHA256 hashes from file %s: %w", arg, err)
	}

	var hashes []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if sha256Re.MatchString(line) {
			hashes = append(hashes, line)
		}
	}
	return hashes, nil
}

func downloadFiles(web webapi.Service, token string, hashes []string) error {
	model := newDownloadModel(hashes, web, token, outputFlag, parallelFlag, extractFlag)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
