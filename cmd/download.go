// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"log"
	"path/filepath"

	"github.com/saferwall/saferwall-cli/internal/util"
	"github.com/saferwall/saferwall-cli/internal/webapi"
	"github.com/spf13/cobra"
)

var sha256Flag string
var outputFlag string

func init() {
	downloadCmd.Flags().StringVarP(&sha256Flag, "hash", "", "",
		"SHA256 hash to download (required)")
	downloadCmd.Flags().StringVarP(&outputFlag, "output", "o", "./",
		"Destination directory where to save samples. (default=current dir)")
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a sample given its SHA256 hash.",
	Long:  `Download a binary sample given a sha256.`,
	Run: func(cmd *cobra.Command, args []string) {

		// load env variable
		username, password := loadEnv()

		// login to saferwall web service
		token, err := webapi.Login(username, password)
		if err != nil {
			log.Fatalf("failed to login to saferwall web service")
		}

		// download the binary
		download(sha256Flag, token, outputFlag)
	},
}

func download(sha256, token, destPath string) error {
	data, err := webapi.Download(sha256, token)
	if err != nil {
		log.Fatalf("failed to download %s, err: %v", sha256, err)
		return err
	}

	filename := sha256 + ".zip"
	destPath = filepath.Join(destPath, filename)

	_, err = util.WriteBytesFile(destPath, data)
	if err != nil {
		log.Fatalf("failed to write bytes to file %s, err: %v", sha256, err)
		return err
	}

	return nil
}
