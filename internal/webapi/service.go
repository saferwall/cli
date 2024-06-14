// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package webapi

const (
	authEndpoint  = "/v1/auth/login/"
	usersEndpoint = "/v1/users/"
	filesEndpoint = "/v1/files/"
)

type Service struct {
	filesURL string
	authURL  string
	usersURL string
}

// New generates new web apis service object.
func New(baseURL string) Service {
	s := Service{}
	s.authURL = baseURL + authEndpoint
	s.usersURL = baseURL + usersEndpoint
	s.filesURL = baseURL + filesEndpoint
	return s
}
