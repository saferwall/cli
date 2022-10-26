// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "saferwall-cli",
	Short: "A cli tool for saferwall.com",
	Long: `saferwall-cli - Saferwall command line tool

	███████╗ █████╗ ███████╗███████╗██████╗ ██╗    ██╗ █████╗ ██╗     ██╗          ██████╗██╗     ██╗
	██╔════╝██╔══██╗██╔════╝██╔════╝██╔══██╗██║    ██║██╔══██╗██║     ██║         ██╔════╝██║     ██║
	███████╗███████║█████╗  █████╗  ██████╔╝██║ █╗ ██║███████║██║     ██║         ██║     ██║     ██║
	╚════██║██╔══██║██╔══╝  ██╔══╝  ██╔══██╗██║███╗██║██╔══██║██║     ██║         ██║     ██║     ██║
	███████║██║  ██║██║     ███████╗██║  ██║╚███╔███╔╝██║  ██║███████╗███████╗    ╚██████╗███████╗██║
	╚══════╝╚═╝  ╚═╝╚═╝     ╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝ ╚═╝  ╚═╝╚══════╝╚══════╝     ╚═════╝╚══════╝╚═╝


saferwall-cli allows you to interact with the saferwall API. You can
upload, scan samples from your drive, or download samples.
For more details see the github repo at https://github.com/saferwall
`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Vesion number",
	Long:  "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("You are using version 0.2.0")
	},
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(genCmd)
}
