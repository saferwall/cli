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

var outputFlag string

func init() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	downloadCmd.Flags().StringVarP(&outputFlag, "output", "o", filepath.Dir(ex),
		"Destination directory where to save samples. (default=current dir)")
}

var downloadCmd = &cobra.Command{
	Use:   "download <sha256|file.txt>",
	Short: "Download a sample (and its artifacts)",
	Long:  `Download a binary sample given a SHA256 hash, or a batch of samples from a text file containing one hash per line.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		arg := args[0]

		// Login to saferwall web service.
		webSvc := webapi.New(cfg.Credentials.URL)
		token, err := webSvc.Login(cfg.Credentials.Username, cfg.Credentials.Password)
		if err != nil {
			log.Fatalf("failed to login to saferwall web service")
		}

		if sha256Re.MatchString(arg) {
			// Single hash: download directly.
			if err := download(arg, token, webSvc); err != nil {
				log.Fatalf("failed to download sample (%s): %v", arg, err)
			}
		} else {
			// Treat as a text file of hashes.
			data, err := util.ReadAll(arg)
			if err != nil {
				log.Fatalf("failed to read SHA256 hashes from file: %s", arg)
			}

			for _, sha256 := range strings.Split(string(data), "\n") {
				sha256 = strings.TrimSpace(sha256)
				if sha256Re.MatchString(sha256) {
					if err := download(sha256, token, webSvc); err != nil {
						log.Fatalf("failed to download sample (%s): %v", sha256, err)
					}
				}
			}
		}
	},
}

func download(sha256, token string, web webapi.Service) error {
	var data bytes.Buffer

	log.Printf("downloading %s to %s", sha256, outputFlag)
	dataContent, err := web.Download(sha256, token)
	if err != nil {
		log.Fatalf("failed to download %s, err: %v", sha256, err)
		return err
	}
	data = *dataContent

	filename := sha256 + ".zip"
	destPath := filepath.Join(outputFlag, filename)
	_, err = util.WriteBytesFile(destPath, &data)
	return err
}
