// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"context"
	"log"

	s "github.com/saferwall/saferwall-cli/internal/storage"
	"github.com/saferwall/saferwall-cli/internal/util"
	"github.com/saferwall/saferwall-cli/internal/webapi"
	"github.com/spf13/cobra"
)

var (
	walkDBFlag bool
	walkS3Flag bool
)

func init() {
	listCmd.AddCommand(listUsersCmd)
	listCmd.AddCommand(listFilesCmd)

	listCmd.PersistentFlags().BoolVarP(&walkDBFlag, "db", "d", false,
		"Paginate over the list of users or files in DB (default: false)")
	listCmd.PersistentFlags().BoolVarP(&walkS3Flag, "s3", "s", false,
		"Paginate over the list of users or files in S3 (default: false)")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List users or files.",
	Long:  `Paginate over the list of users or file in DB or in S3.`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var listUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "List users.",
	Long:  `Paginate over the list of users in DB or in S3.`,
	Run: func(cmd *cobra.Command, args []string) {

		token, err := webapi.Login(cfg.Credentials.Username, cfg.Credentials.Password)
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
	Long:  `Paginate over the list of files in DB or in S3.`,
	Run: func(cmd *cobra.Command, args []string) {

		if walkDBFlag {

			token, err := webapi.Login(cfg.Credentials.Username, cfg.Credentials.Password)
			if err != nil {
				log.Fatalf("failed to login to saferwall web service")
			}

			files, err := webapi.ListFiles(token)
			if err != nil {
				log.Fatalf("failed to list files")
			}

			log.Print(len(files))
		} else if walkS3Flag {
			opts := s.Options{}
			switch cfg.Storage.DeploymentKind {
			case "aws":
				opts.Region = cfg.Storage.S3.Region
				opts.AccessKey = cfg.Storage.S3.AccessKey
				opts.SecretKey = cfg.Storage.S3.SecretKey
			case "minio":
				opts.Region = cfg.Storage.MinIO.Region
				opts.AccessKey = cfg.Storage.MinIO.AccessKey
				opts.SecretKey = cfg.Storage.MinIO.SecretKey
				opts.MinIOEndpoint = cfg.Storage.MinIO.Endpoint
			case "local":
				opts.LocalRootDir = cfg.Storage.Local.RootDir
			}

			opts.Bucket = cfg.Storage.Bucket

			sto, err := s.New(cfg.Storage.DeploymentKind, opts)
			if err != nil {
				log.Fatalf("failed to create a storage client: %v", err)
				return
			}

			files, err := sto.List(context.TODO(), cfg.Storage.Bucket)
			if err != nil {
				log.Fatalf("failed to list objects in bucket: %v", err)
				return
			}

			// Using bytes.Buffer for concatenation to avoid extra mem allocation using +.
			var fileContent bytes.Buffer
			for _, file := range files {
				fileContent.WriteString(file)
				fileContent.WriteString("\n")
			}

			_, err = util.WriteBytesFile("s3-all-docs.txt", &fileContent)
			if err != nil {
				log.Fatalf("failed to write data to file: %v", err)
				return
			}
		}

	},
}
