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
	"net/http"
)

const (
	fileURL = "https://api.saferwall.com/v1/files/"
)

// FileExists determines file existence.
// TODO: use HEAD instead.
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
	jsonBody, err := io.ReadAll(resp.Body)
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

// ListFiles list all the files in DB.
func ListFiles(authToken string, page int) (*Pages, error) {

	var pages Pages
	url := fmt.Sprintf("%s?per_page=%d&page=%d&fields=sha256", fileURL, 1000, page)
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
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var jsonBody map[string]interface{}
		err = json.Unmarshal(body.Bytes(), &jsonBody)
		if err != nil {
			return nil, err
		}
		msg := jsonBody["message"].(string)
		return nil, errors.New(msg)
	}

	err = json.Unmarshal(body.Bytes(), &pages)
	if err != nil {
		return nil, err
	}

	return &pages, nil

}
