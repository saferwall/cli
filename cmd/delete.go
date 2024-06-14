// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"log"
	"strings"

	s "github.com/saferwall/saferwall-cli/internal/storage"
	"github.com/saferwall/saferwall-cli/internal/util"
	"github.com/saferwall/saferwall-cli/internal/webapi"
	"github.com/spf13/cobra"
)

var delFromDB bool
var delFromObjSto bool

func init() {
	deleteCmd.Flags().
		StringVarP(&sha256Flag, "sha256", "s", "",
			"SHA256 hash to delete")
	deleteCmd.Flags().StringVarP(&txtFlag, "txt", "t", "",
		"Delete all hashes in a text file, separate by a line break")
	deleteCmd.Flags().BoolVarP(&delFromDB, "deleteFromDB", "d", false,
		"Delete the sample from the DB")
	deleteCmd.Flags().BoolVarP(&delFromObjSto, "deleteFromStore", "r", false,
		"Delete the sample from the object storage")
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a sample(s) given its SHA256 hash.",
	Long:  `Delete a binary sample given a sha256.`,
	Run: func(cmd *cobra.Command, args []string) {

		svc := webapi.New(cfg.Credentials.URL)

		var token string
		var sto s.Storage
		var err error

		if delFromDB {
			// Authenticate to Saferwall web service.
			token, err = svc.Login(cfg.Credentials.Username, cfg.Credentials.Password)
			if err != nil {
				log.Fatalf("failed to login to saferwall web service")
			}
		}

		if delFromObjSto {
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

			opts.Bucket = cfg.Storage.SamplesBucket

			sto, err = s.New(cfg.Storage.DeploymentKind, opts)
			if err != nil {
				log.Fatalf("failed to create a storage client: %v", err)
				return
			}
		}

		// Delete a single binary.
		if sha256Flag != "" {
			delete(svc, sha256Flag, token, sto)
		} else if txtFlag != "" {
			// Delete a list of sha256 hashes.
			data, err := util.ReadAll(txtFlag)
			if err != nil {
				log.Fatalf("failed to read to SHA256 hashes from txt file: %v", txtFlag)
			}

			sha256list := strings.Split(string(data), "\n")
			for _, sha256 := range sha256list {
				if len(sha256) >= 64 {
					delete(svc, sha256, token, sto)
				}
			}
		}
	},
}

func delete(svc webapi.Service, sha256, token string, sto s.Storage) error {

	log.Printf("deleting %s", sha256)

	if token != "" {
		err := svc.Delete(sha256, token)
		if err != nil {
			log.Fatalf("failed to delete %s, err: %v", sha256, err)
			return err
		}
	}

	if sto != nil {
		err := sto.Delete(context.TODO(), cfg.Storage.SamplesBucket, sha256)
		if err != nil {
			log.Fatalf("failed to delete %s, err: %v", sha256, err)
			return err
		}

	}

	return nil
}
