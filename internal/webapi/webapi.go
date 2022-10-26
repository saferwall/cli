// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package webapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/saferwall/saferwall-cli/internal/entity"
)

const (
	fileURL = "https://api.saferwall.com/v1/files/"
	authURL = "https://api.saferwall.com/v1/auth/login/"
)

func newfileUploadRequest(uri, fieldname, filename string) (*http.Request, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldname, filepath.Base(filename))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err
}

func Login(username, password string) (string, error) {
	requestBody, err := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		return "", err
	}

	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	body := bytes.NewBuffer(requestBody)
	request, err := http.NewRequest(http.MethodPost, authURL, body)
	if err != nil {
		return "", err
	}

	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var res map[string]string
	err = json.Unmarshal(d, &res)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("failed login")
	}

	return res["token"], nil
}

func Upload(filepath string, authToken string) (string, error) {

	// Create a new file upload request.
	request, err := newfileUploadRequest(fileURL, "file", filepath)
	if err != nil {
		return "", err
	}

	// Add our auth token.
	request.Header.Set("Cookie", "JWTCookie="+authToken)

	// Perform the http post request.
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}

	// Read the response.
	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	return body.String(), nil
}

func Rescan(sha256, authToken string) error {

	url := fileURL + sha256 + "/rescan"
	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", "JWTCookie="+authToken)

	// Perform the http post request.
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	// Read the response.
	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		return err
	}

	resp.Body.Close()
	fmt.Println(body)
	return nil
}

// GetFile retrieves the file report given a sha256.
func GetFile(sha256 string, file *entity.File) error {

	url := fileURL + sha256
	client := &http.Client{}
	client.Timeout = time.Second * 10

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(d, &file)
}

func FileExists(sha256 string) (bool, error) {

	url := fileURL + sha256 + "?fields=status"
	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, err
	}

	defer resp.Body.Close()
	jsonBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var file map[string]interface{}
	if err := json.Unmarshal(jsonBody, &file); err != nil {
		return false, err
	}

	if val, ok := file["status"]; ok {
		status := val.(float64)
		if status == 2 {
			return true, nil
		}
	}
	return false, nil
}

func ListFiles(authToken string) ([]string, error) {

	var listSha256 []string
	url := fileURL + "?fields=sha256"
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Cookie", "JWTCookie="+authToken)

	// Perform the http post request.
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	// Read the response.
	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	var shaMap []map[string]string
	err = json.Unmarshal(body.Bytes(), &shaMap)
	if err != nil {
		return nil, err
	}

	for _, v := range shaMap {
		listSha256 = append(listSha256, v["sha256"])
	}

	resp.Body.Close()
	return listSha256, nil
}

func Download(sha256, authToken string) (*bytes.Buffer, error) {

	url := fileURL + sha256 + "/download"
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", "JWTCookie="+authToken)

	// Perform the http post request.
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	// Read the response.
	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return body, nil
}
