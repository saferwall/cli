// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package webapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

// Pages represents a paginated list of data items.
type Pages struct {
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	PageCount  int         `json:"page_count"`
	TotalCount int         `json:"total_count"`
	Items      interface{} `json:"items"`
}

func Login(url, username, password string) (string, error) {

	authURL := url + "/v1/auth/login/"
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
	d, err := io.ReadAll(resp.Body)
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
