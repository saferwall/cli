// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package webapi

import (
	"net/http"
	"time"
)

const (
	filesEndpoint     = "/v1/files/"
	behaviorsEndpoint = "/v1/behaviors/"

	defaultTimeout = 5 * time.Minute
)

type Service struct {
	filesURL     string
	behaviorsURL string
	client       *http.Client
}

// New generates new web apis service object.
func New(baseURL string) Service {
	return Service{
		client:       &http.Client{Timeout: defaultTimeout},
		filesURL:     baseURL + filesEndpoint,
		behaviorsURL: baseURL + behaviorsEndpoint,
	}
}
