// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/saferwall/cli/internal/util"
	"github.com/saferwall/cli/internal/webapi"
)

// Per-file state in the download TUI.
type dlState int

const (
	dlPending      dlState = iota
	dlDownloading          // download in progress
	dlDone                 // download finished successfully
	dlError                // an error occurred
)

// One row in the download UI.
type dlRow struct {
	sha256  string
	state   dlState
	spinner spinner.Model
	dest    string // destination file path (set on success)
	err     error
}

// Top-level bubbletea model for downloads.
type downloadModel struct {
	files    []dlRow
	web      webapi.Service
	token    string
	outDir   string
	parallel int
	done     bool
}

// --- Messages ---

type fileDownloadedMsg struct {
	index int
	dest  string
	err   error
}

// --- Commands (async I/O) ---

func downloadFileCmd(index int, web webapi.Service, sha256, token, outDir string) tea.Cmd {
	return func() tea.Msg {
		dataContent, err := web.Download(sha256, token)
		if err != nil {
			return fileDownloadedMsg{index: index, err: fmt.Errorf("download: %w", err)}
		}

		filename := sha256 + ".zip"
		destPath := filepath.Join(outDir, filename)
		if _, err := util.WriteBytesFile(destPath, dataContent); err != nil {
			return fileDownloadedMsg{index: index, err: fmt.Errorf("write file: %w", err)}
		}

		return fileDownloadedMsg{index: index, dest: destPath}
	}
}

// --- Model interface ---

func newDownloadModel(hashes []string, web webapi.Service, token, outDir string, parallel int) downloadModel {
	if parallel < 1 {
		parallel = 1
	}
	rows := make([]dlRow, len(hashes))
	for i, h := range hashes {
		s := spinner.New()
		s.Spinner = spinner.Dot
		rows[i] = dlRow{
			sha256:  h,
			state:   dlPending,
			spinner: s,
		}
	}
	return downloadModel{
		files:    rows,
		web:      web,
		token:    token,
		outDir:   outDir,
		parallel: parallel,
	}
}

func (m downloadModel) Init() tea.Cmd {
	if len(m.files) == 0 {
		return tea.Quit
	}

	n := min(m.parallel, len(m.files))
	var cmds []tea.Cmd
	for i := range n {
		m.files[i].state = dlDownloading
		cmds = append(cmds,
			downloadFileCmd(i, m.web, m.files[i].sha256, m.token, m.outDir),
			m.files[i].spinner.Tick,
		)
	}
	return tea.Batch(cmds...)
}

func (m downloadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		for i := range m.files {
			if m.files[i].state == dlDownloading {
				var cmd tea.Cmd
				m.files[i].spinner, cmd = m.files[i].spinner.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case fileDownloadedMsg:
		i := msg.index
		if msg.err != nil {
			m.files[i].state = dlError
			m.files[i].err = msg.err
		} else {
			m.files[i].state = dlDone
			m.files[i].dest = msg.dest
		}
		return m, m.maybeQuitOrNext()
	}

	return m, tea.Batch(cmds...)
}

// maybeQuitOrNext launches pending downloads up to the parallel limit, or quits if all done.
func (m *downloadModel) maybeQuitOrNext() tea.Cmd {
	inFlight := 0
	allDone := true
	for _, f := range m.files {
		switch f.state {
		case dlDownloading:
			inFlight++
			allDone = false
		case dlPending:
			allDone = false
		}
	}
	if allDone {
		m.done = true
		return tea.Quit
	}

	var cmds []tea.Cmd
	for i := range m.files {
		if inFlight >= m.parallel {
			break
		}
		if m.files[i].state == dlPending {
			m.files[i].state = dlDownloading
			cmds = append(cmds,
				downloadFileCmd(i, m.web, m.files[i].sha256, m.token, m.outDir),
				m.files[i].spinner.Tick,
			)
			inFlight++
		}
	}
	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}
	return nil
}

func (m downloadModel) View() string {
	var s string
	for _, f := range m.files {
		sha := truncSha(f.sha256)
		switch f.state {
		case dlPending:
			s += styleDim.Render("  "+sha) + "\n"

		case dlDownloading:
			s += f.spinner.View() + styleLabel.Render(" Downloading ") + sha + " ...\n"

		case dlDone:
			s += styleSuccess.Render("✓") + " " + sha + "  " + styleDim.Render(f.dest) + "\n"

		case dlError:
			s += styleError.Render("✗") + " " + sha + "  " + styleError.Render(f.err.Error()) + "\n"
		}
	}
	return s
}
