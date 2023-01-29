// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package webapi

import (
	"bytes"
	"encoding/json"
	"net/http"
)

const (
	usersURL = "https://api.saferwall.com/v1/users/"
)

// ListUsers returns the list of users.
func ListUsers(authToken string) ([]string, error) {

	request, err := http.NewRequest("GET", usersURL, nil)
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

	var usersList []map[string]interface{}
	err = json.Unmarshal(body.Bytes(), &usersList)
	if err != nil {
		return nil, err
	}

	var usernamesList []string
	for _, user := range usersList {
		usernamesList = append(usernamesList, user["username"].(string))
	}

	resp.Body.Close()
	return usernamesList, nil
}
