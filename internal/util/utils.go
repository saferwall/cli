// Copyright 2022 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package util

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// GetSha256 returns SHA256 hash.
func GetSha256(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

// ReadAll reads the entire file into memory.
func ReadAll(filePath string) ([]byte, error) {
	// Start by getting a file descriptor over the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Get the file size to know how much we need to allocate
	fileinfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	filesize := fileinfo.Size()
	buffer := make([]byte, filesize)

	// Read the whole binary
	_, err = file.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

// WriteBytesFile write Bytes to a File.
func WriteBytesFile(filename string, r io.Reader) (int, error) {

	// Open a new file for writing only
	file, err := os.OpenFile(
		filename,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Read for the reader bytes to file
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return 0, err
	}

	// Write bytes to disk
	bytesWritten, err := file.Write(b)
	if err != nil {
		return 0, err
	}

	return bytesWritten, nil
}

// Exists reports whether the named file or directory exists.
func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// MkDir create a directory if it does not exists.
func MkDir(name string) bool {
	if !Exists(name) {
		return os.Mkdir(name, 0755) == nil
	}
	return true
}

// StringInSlice returns whether or not a string exists in a slice.
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// UniqueSlice delete duplicate strings from an array of strings.
func UniqueSlice(slice []string) []string {
	cleaned := []string{}

	for _, value := range slice {
		if !StringInSlice(value, cleaned) {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

// WalkAllFilesInDir returns list of files in directory.
func WalkAllFilesInDir(dir string) ([]string, error) {

	fileList := []string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}

		// check if it is a regular file (not dir)
		if info.Mode().IsRegular() {
			fileList = append(fileList, path)
		}
		return nil
	})

	return fileList, err
}

func UserHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}
