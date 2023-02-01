// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"log"

	"github.com/saferwall/saferwall-cli/internal/webapi"
	"github.com/spf13/cobra"
)

func init() {
	listCmd.AddCommand(listUsersCmd)
	listCmd.AddCommand(listFilesCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List users or files.",
	Long:  `Paginate over the list of users or file in DB.`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var listUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "List users.",
	Long:  `Paginate over the list of users in DB.`,
	Run: func(cmd *cobra.Command, args []string) {

		username, password := loadEnv()

		token, err := webapi.Login(username, password)
		if err != nil {
			log.Fatalf("failed to login to saferwall web service")
		}

		users, err := webapi.ListUsers(token)
		if err != nil {
			log.Fatalf("failed to list users")
		}

		log.Print(len(users))
	},
}

var listFilesCmd = &cobra.Command{
	Use:   "files",
	Short: "List files.",
	Long:  `Paginate over the list of files in DB.`,
	Run: func(cmd *cobra.Command, args []string) {
		// load env variable
		username, password := loadEnv()

		// login to saferwall web service
		token, err := webapi.Login(username, password)
		if err != nil {
			log.Fatalf("failed to login to saferwall web service")
		}

		files, err := webapi.ListFiles(token)
		if err != nil {
			log.Fatalf("failed to list files")
		}

		log.Print(len(files))
	},
}
