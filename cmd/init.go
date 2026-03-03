// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saferwall/cli/internal/util"
	"github.com/spf13/cobra"
)

const configTemplate = `[credentials]
# The URL used to interact with saferwall APIs.
url = %q
# The user name you choose when you sign-up for saferwall.com.
username = %q
# The password you choose when you sign-up for saferwall.com.
password = %q
`

func configDir() string {
	return filepath.Join(util.UserHomeDir(), ".config", "saferwall")
}

func configPath() string {
	return filepath.Join(configDir(), "config.toml")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Configure saferwall CLI credentials",
	Long:  `Interactively configure the credentials used to access the Saferwall web API. Creates ~/.config/saferwall/config.toml.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		path := configPath()

		if _, err := os.Stat(path); err == nil {
			log.Fatalf("config already exists: %s\nDelete it first to reinitialize.", path)
		}

		model := newInitModel()
		p := tea.NewProgram(model)
		result, err := p.Run()
		if err != nil {
			log.Fatalf("TUI error: %v", err)
		}

		m := result.(initModel)
		if !m.submitted {
			fmt.Println("Cancelled.")
			return
		}

		url, username, password := m.values()
		if username == "" || password == "" {
			log.Fatalf("username and password are required")
		}

		if err := os.MkdirAll(configDir(), 0o700); err != nil {
			log.Fatalf("failed to create config directory: %v", err)
		}

		content := fmt.Sprintf(configTemplate, url, username, password)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			log.Fatalf("failed to write config file: %v", err)
		}

		fmt.Println(styleSuccess.Render("✓") + " Config written to " + path)
	},
}
