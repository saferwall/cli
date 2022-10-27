// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package entity

type Sample struct {
	SHA256      string `yaml:"sha256"`
	FileFormat  string `yaml:"fileformat"`
	Category    string `yaml:"category"`
	Platform    string `yaml:"platform"`
	Description string `yaml:"description"`
}

type Family struct {
	Name       string   `yaml:"name"`
	Aliases    []string `yaml:"aliases"`
	FirstSeen  string   `yaml:"first_seen"`
	References []string `yaml:"references"`
	Samples    []Sample `yaml:"samples"`
}
