// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package webapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/saferwall/saferwall-cli/internal/entity"
)

const (
	usersURL = "https://api.saferwall.com/v1/users/"
)

// ListUsers returns the list of users.
func ListUsers(authToken string) ([]entity.User, error) {

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

	pages := Pages{}
	err = json.Unmarshal(body.Bytes(), &pages)
	if err != nil {
		return nil, err
	}

	users := []entity.User{}
	for page := 1; page <= pages.PageCount; page++ {
		newUsers, err := ListUsersWithIndex(authToken, page, pages.PerPage)
		if err != nil {
			return nil, err
		}
		users = append(users, newUsers...)

	}

	return users, nil
}

// ListUsers returns the list of users given a page and a per-page data.
func ListUsersWithIndex(authToken string, page, perPage int) ([]entity.User, error) {

	url := fmt.Sprintf("%s?page=%d&perPage=%d", usersURL, page, perPage)
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

	pages := Pages{}
	err = json.Unmarshal(body.Bytes(), &pages)
	if err != nil {
		return nil, err
	}

	usersMap := pages.Items.([]interface{})

	var users []entity.User
	for _, u := range usersMap {
		var user entity.User
		data, _ := json.Marshal(u)
		json.Unmarshal(data, &user)
		users = append(users, user)
	}

	resp.Body.Close()
	return users, nil
}
