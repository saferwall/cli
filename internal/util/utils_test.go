// Copyright 2022 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package util

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGetSha256(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "empty input",
			input: []byte{},
			want:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:  "known vector",
			input: []byte("abc"),
			want:  "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSha256(tt.input); got != tt.want {
				t.Errorf("GetSha256() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestReadAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.bin")
	content := bytes.Repeat([]byte("saferwall"), 1000)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("ReadAll() returned %d bytes, want %d", len(got), len(content))
	}
}

func TestReadAllMissingFile(t *testing.T) {
	if _, err := ReadAll(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("ReadAll() expected error for missing file, got nil")
	}
}

func TestWriteBytesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.bin")
	content := []byte("hello saferwall")

	n, err := WriteBytesFile(path, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("WriteBytesFile() error = %v", err)
	}
	if n != len(content) {
		t.Errorf("WriteBytesFile() wrote %d bytes, want %d", n, len(content))
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("file content = %q, want %q", got, content)
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	if !Exists(path) {
		t.Errorf("Exists(%q) = false, want true", path)
	}
	if !Exists(dir) {
		t.Errorf("Exists(%q) = false, want true for directory", dir)
	}
	if Exists(filepath.Join(dir, "missing")) {
		t.Error("Exists() = true for missing file, want false")
	}
}

func TestMkDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "newdir")

	if !MkDir(dir) {
		t.Fatalf("MkDir(%q) = false, want true", dir)
	}
	if !Exists(dir) {
		t.Errorf("MkDir(%q) did not create the directory", dir)
	}
	// Calling it again on an existing directory succeeds.
	if !MkDir(dir) {
		t.Errorf("MkDir(%q) = false on existing directory, want true", dir)
	}
}

func TestStringInSlice(t *testing.T) {
	list := []string{"a", "b", "c"}
	if !StringInSlice("b", list) {
		t.Error(`StringInSlice("b") = false, want true`)
	}
	if StringInSlice("z", list) {
		t.Error(`StringInSlice("z") = true, want false`)
	}
	if StringInSlice("a", nil) {
		t.Error("StringInSlice() on nil slice = true, want false")
	}
}

func TestUniqueSlice(t *testing.T) {
	got := UniqueSlice([]string{"a", "b", "a", "c", "b"})
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("UniqueSlice() = %v, want %v", got, want)
	}
}

func TestWalkAllFilesInDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{
		filepath.Join(dir, "a.txt"),
		filepath.Join(sub, "b.txt"),
	} {
		if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := WalkAllFilesInDir(dir)
	if err != nil {
		t.Fatalf("WalkAllFilesInDir() error = %v", err)
	}
	want := []string{filepath.Join(dir, "a.txt"), filepath.Join(sub, "b.txt")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("WalkAllFilesInDir() = %v, want %v", got, want)
	}
}
