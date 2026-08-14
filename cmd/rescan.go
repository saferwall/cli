// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saferwall/cli/internal/util"
	"github.com/saferwall/cli/internal/webapi"
	"github.com/spf13/cobra"
)

var sha256Re = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

var (
	rescanNetworkEnabled bool
	rescanCountry        string
)

func init() {
	reScanCmd.Flags().IntVar(&parallelFlag, "parallel", 1,
		"Number of files to rescan in parallel")
	reScanCmd.Flags().BoolVarP(&enableDetonationFlag, "enableDetonation", "d", false,
		"Enable sandbox detonation (skipped by default)")
	reScanCmd.Flags().IntVarP(&timeoutFlag, "timeout", "t", defaultDetonationTimeout,
		"Detonation duration in seconds")
	reScanCmd.Flags().StringVarP(&osFlag, "os", "o", "windows-10-x64",
		"Preferred OS for detonation, choice(windows-7-x64 | windows-10-x64 | windows-11-x64)")
	reScanCmd.Flags().BoolVar(&rescanNetworkEnabled, "network", true,
		"Allow sandbox internet access")
	reScanCmd.Flags().StringVar(&rescanCountry, "country", "US",
		"Two-letter VPN exit country when network access is enabled")
}

// reScanFile re-scans a list of SHA256 with a TUI progress display.
func reScanFile(web webapi.Service, shaList []string, token string) error {
	model := newRescanModel(shaList, web, token, parallelFlag)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

var reScanCmd = &cobra.Command{
	Use:   "rescan <sha256|file>",
	Short: "Rescan an existing file using its hash",
	Long:  `Rescans one or more files. Pass a SHA256 hash to rescan a single file, or a path to a text file with one hash per line to rescan in batch.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		webSvc := webapi.New(cfg.Credentials.URL)
		arg := args[0]

		var sha256List []string
		if sha256Re.MatchString(arg) {
			sha256List = append(sha256List, arg)
		} else {
			data, err := util.ReadAll(arg)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", arg, err)
			}
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					sha256List = append(sha256List, line)
				}
			}
		}

		return reScanFile(webSvc, sha256List, cfg.Credentials.APIKey)
	},
}
