// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"context"
	"log"
	"path/filepath"

	s "github.com/saferwall/saferwall-cli/internal/storage"
	"github.com/saferwall/saferwall-cli/internal/util"
	"github.com/spf13/cobra"
)

func init() {
	uploadCmd.Flags().StringVarP(&filePath, "path", "p", "",
		"Destination directory for the file to upload.")
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload samples directly to object storage.",
	Long:  `Uploading samples directly to object storage and without triggering a scan`,
	Run: func(cmd *cobra.Command, args []string) {

		// Authenticate to object storage service.
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

		filePaths, err := util.WalkAllFilesInDir(filePath)
		if err != nil {
			log.Fatalf("failed to walk directory %s: %v", filePaths, err)
		}

		for _, filePath := range filePaths {
			fileContent, err := util.ReadAll(filePath)
			if err != nil {
				log.Fatalf("failed to read file %s,: %v", filePath, err)
			}
			sha256 := filepath.Base(filePath)
			r := bytes.NewReader(fileContent)

			log.Printf("uploading %s", sha256)
			err = sto.Upload(context.TODO(), cfg.Storage.Bucket, sha256, r)
			if err != nil {
				log.Fatalf("failed to upload file %s: %v", filePath, err)
			}
		}

	},
}
