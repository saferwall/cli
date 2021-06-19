package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	sfwcli "github.com/saferwall/saferwall-cli"
)

const (
	bucket  = "saferwall-samples"
	region  = "us-east-1"
	timeout = 0
)

var (
	forceRescan bool
	outputDir   string
	username    string
	password    string
)

// scanFile scans an individual file or a directory.
func scanFile(cmd *cobra.Command, args []string) {

	pathToScan := args[0]
	_, err := os.Stat(pathToScan)
	if os.IsNotExist(err) {
		log.Fatalf("%s does not exist", pathToScan)
	}

	// Walk over directory.
	fileList := []string{}
	filepath.Walk(pathToScan, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			fileList = append(fileList, path)
		}
		return nil
	})

	// Obtain a token.
	token, err := sfwcli.Login(username, password)
	if err != nil {
		log.Fatal("API authentification failed with error :", err)
		return
	}

	// Upload files
	for _, filename := range fileList {

		// Get sha256
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatal("readfile failed with error :", err)
			return
		}
		sha256 := sfwcli.SHA256(data)

		// Do we have the file in S3.
		if !sfwcli.IsFileFoundInObjStorage(sha256) {
			sfwcli.UploadObject(bucket, region, sha256, filename)
		}

		// Check if we the file exists in the DB.
		if !sfwcli.IsFileFoundInDB(sha256, token) {
			sfwcli.Scan(sha256, token)
			time.Sleep(timeout * time.Second)
			continue
		}

		// Force rescan the file?.
		if forceRescan {
			sfwcli.Rescan(sha256, token)
			time.Sleep(timeout * time.Second)
		}
	}
}

func s3upload(cmd *cobra.Command, args []string) {

	filePath := args[0]
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		sfwcli.ExitWithError("%s does not exist", filePath)
	}

	objKeys := sfwcli.ListObject(bucket, region, false)

	// Walk over directory.
	fileList := []string{}
	filepath.Walk(filePath, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			fileList = append(fileList, path)
		}
		return nil
	})

	// Upload files
	for _, filename := range fileList {
		// Check if we have the file already in our database.
		dat, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Printf("failed to read file %s", filename)
			continue
		}
		key := sfwcli.SHA256(dat)
		found := sfwcli.StringInSlice(key, objKeys)
		if !found {
			sfwcli.UploadObject(bucket, region, key, filename)
		} else {
			fmt.Printf("file name %s already in s3 bucket", filename)
		}
	}

}

// rescanFile reads a list of sha256 from the clipboard and trigger a rescan.
func rescanFile(cmd *cobra.Command, args []string) {

	// Obtain a token.
	token, err := sfwcli.Login(username, password)
	if err != nil {
		log.Fatal(err)
	}

	clipContent, err := clipboard.ReadAll()
	if err != nil {
		log.Fatal("rescan failed with error : ", err)
	}

	shaList := strings.Split(clipContent, "\r\n")
	for _, sha256 := range shaList {
		sfwcli.Rescan(sha256, token)

		// Wait for file to be scanned.
		time.Sleep(timeout * time.Second)
	}

}

// processAuthTokens processes a given username and password as env variables.
func processAuthTokens() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	username := os.Getenv(sfwcli.DefaultAuthUsername)
	password := os.Getenv(sfwcli.DefaultAuthPassword)
	err = sfwcli.SetAuthentificationData(username, password)
	if err != nil {
		log.Fatal("API authentification failed with error :", err)
	}
}

// downloadFile a list of sha256 from the clipboard and download them.
func downloadFile(cmd *cobra.Command, args []string) {

	clipContent, err := clipboard.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	shaList := strings.Split(clipContent, "\r\n")
	for _, sha256 := range shaList {
		// Create a new file.
		filePath := path.Join(outputDir, sha256)
		f, err := os.Create(filePath)
		if err != nil {
			log.Fatal("failed to create new file with error :", err)
		}
		defer f.Close()

		err = sfwcli.DownloadObject(bucket, region, sha256, f)
		if err != nil {
			log.Println(err)
			continue
		}

	}
}

func main() {

	var rootCmd = &cobra.Command{
		Use:   "saferwall-cli",
		Short: "A cli tool for saferwall.com",
		Long:  sfwcli.WelcomeMessage,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Vesion number",
		Long:  "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print("You are using version 0.2.0")
		},
	}

	var scanCmd = &cobra.Command{
		Use:   "scan",
		Short: "Scan file",
		Long:  "Scan a file or directory",
		Args:  cobra.MinimumNArgs(1),
		Run:   scanFile,
	}

	var rescanCmd = &cobra.Command{
		Use:   "rescan",
		Short: "Resccan file",
		Long:  "Rescan a file or directory",
		Run:   rescanFile,
	}

	var s3UploadCmd = &cobra.Command{
		Use:   "s3upload",
		Short: "S3 upload",
		Long:  "Batch upload to S3",
		Args:  cobra.MinimumNArgs(1),
		Run:   s3upload,
	}

	var downloadCmd = &cobra.Command{
		Use:   "download",
		Short: "Download file",
		Long:  "Download a file or directory",
		Run:   downloadFile,
	}

	// Init root command.
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(rescanCmd)
	rootCmd.AddCommand(s3UploadCmd)
	rootCmd.AddCommand(downloadCmd)

	// Init flags
	scanCmd.Flags().BoolVarP(&forceRescan, "forcerescan", "f", false, "Force rescan the file.")
	downloadCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory to download the files (required")
	downloadCmd.MarkFlagRequired("output")

	// load config
	processAuthTokens()
	// Get credentials.
	username = os.Getenv(sfwcli.DefaultAuthUsername)
	password = os.Getenv(sfwcli.DefaultAuthPassword)
	if username == "" || password == "" {
		fmt.Println("SAFERWALL_AUTH_USERNAME or SAFERWALL_AUTH_PASSWORD env variable are not set.")
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}

}
