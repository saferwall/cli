// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"context"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	s "github.com/saferwall/saferwall-cli/internal/storage"
	"github.com/saferwall/saferwall-cli/internal/util"
	"github.com/saferwall/saferwall-cli/internal/webapi"
	"github.com/spf13/cobra"
)

var sha256Flag string
var bhvReportFlag string
var txtFlag string
var outputFlag string
var useWebAPIs bool

func init() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	downloadCmd.Flags().
		StringVarP(&sha256Flag, "hash", "s", "",
			"SHA256 hash to download")
	downloadCmd.Flags().
		StringVarP(&bhvReportFlag, "behavior-report", "b", "",
			"GUID for behavior report to download")
	downloadCmd.Flags().StringVarP(&txtFlag, "txt", "t", "",
		"Download all hashes in a text file, separate by a line break")
	downloadCmd.Flags().StringVarP(&outputFlag, "output", "o", filepath.Dir(ex),
		"Destination directory where to save samples. (default=current dir)")
	downloadCmd.Flags().BoolVarP(&useWebAPIs, "useWebAPIs", "u", false,
		"Use the web APIs to download samples")
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a sample(s) or a behavior report",
	Long:  `Download a binary sample given a sha256 or a behavior report using its GUID.`,
	Run: func(cmd *cobra.Command, args []string) {

		var token string
		var sto s.Storage
		var err error

		if useWebAPIs {
			// Authenticate to Saferwall web service.
			token, err = webapi.Login(cfg.Credentials.Username, cfg.Credentials.Password)
			if err != nil {
				log.Fatalf("failed to login to saferwall web service")
			}
		} else {
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

		// download a single binary.
		if sha256Flag != "" {
			download(sha256Flag, token, sto)
		} else if txtFlag != "" {
			// Download a list of sha256 hashes.
			data, err := util.ReadAll(txtFlag)
			if err != nil {
				log.Fatalf("failed to read to SHA256 hashes from txt file: %v", txtFlag)
			}

			sha256list := strings.Split(string(data), "\n")
			for _, sha256 := range sha256list {
				if len(sha256) >= 64 {
					err = download(sha256, token, sto)
					if err != nil {
						log.Fatalf("failed to download sample (%s): %v", sha256, err)
					}

				}
			}
		} else if bhvReportFlag != "" {
			err = downloadBhvReport(sto, bhvReportFlag)
			if err != nil {
				log.Fatalf("failed to download behavior report (%s): %v", bhvReportFlag, err)
			}
		}
	},
}

func download(sha256, token string, sto s.Storage) error {

	var err error
	var data bytes.Buffer
	var destPath string

	log.Printf("downloading %s to %s", sha256, outputFlag)

	if token != "" {
		dataContent, err := webapi.Download(sha256, token)
		if err != nil {
			log.Fatalf("failed to download %s, err: %v", sha256, err)
			return err
		}
		data = *dataContent

		filename := sha256 + ".zip"
		destPath = filepath.Join(outputFlag, filename)

	} else {
		err := sto.Download(context.TODO(), cfg.Storage.SamplesBucket, sha256, &data)
		if err != nil {
			return err
		}

		destPath = filepath.Join(outputFlag, sha256)
	}
	_, err = util.WriteBytesFile(destPath, &data)
	if err != nil {
		return err
	}

	return nil
}

func downloadBhvReport(sto s.Storage, bhvReportID string) error {
	var data bytes.Buffer

	log.Printf("downloading behavior report for GUID: %s", bhvReportID)

	objects, err := sto.List(context.TODO(), cfg.Storage.ArtifactsBucket, bhvReportID)
	if err != nil {
		return err
	}

	for _, obj := range objects {
		if strings.HasSuffix(obj, "/") {
			continue
		}
		log.Printf("downloading %s ..", obj)
		err := sto.Download(context.TODO(), cfg.Storage.ArtifactsBucket, obj, &data)
		if err != nil {
			return err
		}

		objPath := filepath.Join(outputFlag, obj)
		os.MkdirAll(path.Dir(objPath), os.ModePerm)
		_, err = util.WriteBytesFile(objPath, &data)
		if err != nil {
			return err
		}

	}

	return nil
}
