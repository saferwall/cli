// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/saferwall/cli/internal/util"
	"github.com/saferwall/cli/internal/webapi"
	"github.com/spf13/cobra"
)

var sha256Flag string
var txtFlag string
var outputFlag string

func init() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	downloadCmd.Flags().StringVarP(&sha256Flag, "hash", "s", "", "SHA256 hash to download")
	downloadCmd.Flags().StringVarP(&txtFlag, "txt", "t", "", "Download all hashes in a text file, separate by a line break")
	downloadCmd.Flags().StringVarP(&outputFlag, "output", "o", filepath.Dir(ex),
		"Destination directory where to save samples. (default=current dir)")
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a sample(s)",
	Long:  `Download a binary sample given a sha256`,
	Run: func(cmd *cobra.Command, args []string) {

		// Login to saferwall web service
		webSvc := webapi.New(cfg.Credentials.URL)
		token, err := webSvc.Login(cfg.Credentials.Username, cfg.Credentials.Password)
		if err != nil {
			log.Fatalf("failed to login to saferwall web service")
		}

		// download a single binary.
		if sha256Flag != "" {
			download(sha256Flag, token, webSvc)
		} else if txtFlag != "" {
			// Download a list of sha256 hashes.
			data, err := util.ReadAll(txtFlag)
			if err != nil {
				log.Fatalf("failed to read to SHA256 hashes from txt file: %v", txtFlag)
			}

			sha256list := strings.Split(string(data), "\n")
			for _, sha256 := range sha256list {
				if len(sha256) >= 64 {
					err = download(sha256, token, webSvc)
					if err != nil {
						log.Fatalf("failed to download sample (%s): %v", sha256, err)
					}
				}
			}
		}
	},
}

func download(sha256, token string, web webapi.Service) error {
	var err error
	var data bytes.Buffer
	var destPath string

	log.Printf("downloading %s to %s", sha256, outputFlag)
	dataContent, err := web.Download(sha256, token)
	if err != nil {
		log.Fatalf("failed to download %s, err: %v", sha256, err)
		return err
	}
	data = *dataContent

	filename := sha256 + ".zip"
	destPath = filepath.Join(outputFlag, filename)
	_, err = util.WriteBytesFile(destPath, &data)
	if err != nil {
		return err
	}

	return nil
}
