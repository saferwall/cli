// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package webapi

import (
	"net/http"
	"time"
)

const (
	authEndpoint  = "/v1/auth/login/"
	usersEndpoint = "/v1/users/"
	filesEndpoint = "/v1/files/"

	defaultTimeout = 60 * time.Second
)

type Service struct {
	filesURL string
	authURL  string
	usersURL string
	client   *http.Client
}

// New generates new web apis service object.
func New(baseURL string) Service {
	s := Service{
		client: &http.Client{Timeout: defaultTimeout},
	}
	s.authURL = baseURL + authEndpoint
	s.usersURL = baseURL + usersEndpoint
	s.filesURL = baseURL + filesEndpoint
	return s
}
