// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"log"

	"github.com/saferwall/saferwall-cli/internal/entity"
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

			var fileContent bytes.Buffer

			token, err := webapi.Login(cfg.Credentials.Username, cfg.Credentials.Password)
			if err != nil {
				log.Fatalf("failed to login to saferwall web service")
			}

			// Do an initial call to get the page's count.
			pages, err := webapi.ListFiles(token, 1)
			if err != nil {
				log.Fatalf("failed to list files: %v", err)
			}
			log.Printf("the database contains %d files", pages.TotalCount)

			// Iterate over each File page.
			for page := 1; page <= pages.PageCount; page++ {

				log.Printf("getting files for page: %d", page)
				results, err := webapi.ListFiles(token, page)
				if err != nil {
					log.Fatalf("failed to list files: %v", err)
				}

				var listSha256 []string
				files := results.Items.([]interface{})
				for _, fileIf := range files {
					file := entity.File{}
					b, _ := json.Marshal(fileIf)
					json.Unmarshal(b, &file)
					listSha256 = append(listSha256, file.SHA256)
				}

				// Using bytes.Buffer for concatenation to avoid extra mem allocation using +.
				for _, sha256 := range listSha256 {
					fileContent.WriteString(sha256)
					fileContent.WriteString("\n")
				}

			}
			_, err = util.WriteBytesFile("db-all-sha256.txt", &fileContent)
			if err != nil {
				log.Fatalf("failed to write data to file: %v", err)
				return
			}
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

			opts.Bucket = cfg.Storage.SamplesBucket

			sto, err := s.New(cfg.Storage.DeploymentKind, opts)
			if err != nil {
				log.Fatalf("failed to create a storage client: %v", err)
				return
			}

			files, err := sto.List(context.TODO(), cfg.Storage.SamplesBucket, "")
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

			_, err = util.WriteBytesFile("s3-all-sha256.txt", &fileContent)
			if err != nil {
				log.Fatalf("failed to write data to file: %v", err)
				return
			}
		}

	},
}
