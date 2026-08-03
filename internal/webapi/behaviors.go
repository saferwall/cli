// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package webapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GetBehaviorReport fetches selected fields of a behavior document.
// fields must be non-empty: an unfiltered GET inlines the entire API trace,
// which can be enormous.
func (s Service) GetBehaviorReport(id string, fields []string, out any) error {
	if len(fields) == 0 {
		return fmt.Errorf("fields must not be empty")
	}

	query := url.Values{}
	query.Set("fields", strings.Join(fields, ","))
	reqURL := s.behaviorsURL + id + "/?" + query.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	body, err := s.do(req)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// CountSysEvents returns the total number of system events of the given
// type (file, registry or network) recorded for a behavior report, using a
// minimal pagination probe.
func (s Service) CountSysEvents(id, eventType string) (int, error) {
	query := url.Values{}
	query.Set("type", eventType)
	query.Set("page", "1")
	query.Set("per_page", "1")
	reqURL := s.behaviorsURL + id + "/sys-events/?" + query.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, err
	}

	body, err := s.do(req)
	if err != nil {
		return 0, err
	}

	var pages Pages
	if err := json.Unmarshal(body, &pages); err != nil {
		return 0, err
	}
	return pages.TotalCount, nil
}
