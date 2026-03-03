// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/saferwall/cli/internal/entity"
	"github.com/saferwall/cli/internal/util"
	"github.com/saferwall/cli/internal/webapi"
)

// Per-file state in the TUI.
type fileState int

const (
	statePending   fileState = iota
	stateUploading           // upload in progress
	stateScanning            // polling for scan completion
	stateDone                // scan finished successfully
	stateError               // an error occurred
)

const maxPollRetries = 120 // 120 * 5s = 10 minutes

// One row in the UI.
type fileRow struct {
	filename  string
	sha256    string
	state     fileState
	spinner   spinner.Model
	result    *scanSummary
	err       error
	pollCount int
}

// Top-level bubbletea model.
type scanModel struct {
	files    []fileRow
	web      webapi.Service
	token    string
	parallel int
	done     bool
	isRescan bool // true when running in rescan mode (no upload, just rescan API + poll)
}

// --- Messages ---

type fileUploadedMsg struct {
	index  int
	sha256 string
	err    error
}

type fileScanStatusMsg struct {
	index  int
	status int
	err    error
}

type fileScanDoneMsg struct {
	index   int
	summary scanSummary
	err     error
}

// --- Commands (async I/O) ---

func uploadFileCmd(index int, web webapi.Service, filename, token string) tea.Cmd {
	return func() tea.Msg {
		data, err := os.ReadFile(filename)
		if err != nil {
			return fileUploadedMsg{index: index, err: fmt.Errorf("read file: %w", err)}
		}
		sha256 := util.GetSha256(data)

		exists, err := web.FileExists(sha256)
		if err != nil {
			return fileUploadedMsg{index: index, err: fmt.Errorf("check existence: %w", err)}
		}

		if !exists {
			_, err = web.Scan(filename, token, osFlag, enableDetonationFlag, timeoutFlag)
			if err != nil {
				return fileUploadedMsg{index: index, err: fmt.Errorf("upload: %w", err)}
			}
		} else if forceRescanFlag {
			err = web.Rescan(sha256, token, osFlag, enableDetonationFlag, timeoutFlag)
			if err != nil {
				return fileUploadedMsg{index: index, err: fmt.Errorf("rescan: %w", err)}
			}
		}

		return fileUploadedMsg{index: index, sha256: sha256}
	}
}

func pollStatusCmd(index int, web webapi.Service, sha256 string) tea.Cmd {
	return func() tea.Msg {
		status, err := web.GetFileStatus(sha256)
		if err != nil {
			return fileScanStatusMsg{index: index, err: fmt.Errorf("poll status: %w", err)}
		}
		return fileScanStatusMsg{index: index, status: status}
	}
}

func fetchResultCmd(index int, web webapi.Service, sha256 string) tea.Cmd {
	return func() tea.Msg {
		var file entity.File
		if err := web.GetFile(sha256, &file); err != nil {
			return fileScanDoneMsg{index: index, err: fmt.Errorf("get file report: %w", err)}
		}
		summary := buildScanSummary(file)
		return fileScanDoneMsg{index: index, summary: summary}
	}
}

func delayedPollCmd(index int, web webapi.Service, sha256 string) tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg {
		status, err := web.GetFileStatus(sha256)
		if err != nil {
			return fileScanStatusMsg{index: index, err: fmt.Errorf("poll status: %w", err)}
		}
		return fileScanStatusMsg{index: index, status: status}
	})
}

func rescanFileCmd(index int, web webapi.Service, sha256, token string) tea.Cmd {
	return func() tea.Msg {
		err := web.Rescan(sha256, token, osFlag, enableDetonationFlag, timeoutFlag)
		if err != nil {
			return fileUploadedMsg{index: index, err: fmt.Errorf("rescan: %w", err)}
		}
		return fileUploadedMsg{index: index, sha256: sha256}
	}
}

// --- Model interface ---

func newScanModel(files []string, web webapi.Service, token string, parallel int) scanModel {
	if parallel < 1 {
		parallel = 1
	}
	rows := make([]fileRow, len(files))
	for i, f := range files {
		s := spinner.New()
		s.Spinner = spinner.Dot
		rows[i] = fileRow{
			filename: f,
			state:    statePending,
			spinner:  s,
		}
	}
	return scanModel{
		files:    rows,
		web:      web,
		token:    token,
		parallel: parallel,
	}
}

func newRescanModel(sha256List []string, web webapi.Service, token string, parallel int) scanModel {
	if parallel < 1 {
		parallel = 1
	}
	rows := make([]fileRow, len(sha256List))
	for i, sha := range sha256List {
		s := spinner.New()
		s.Spinner = spinner.Dot
		rows[i] = fileRow{
			filename: sha,
			sha256:   sha,
			state:    statePending,
			spinner:  s,
		}
	}
	return scanModel{
		files:    rows,
		web:      web,
		token:    token,
		parallel: parallel,
		isRescan: true,
	}
}

func (m scanModel) Init() tea.Cmd {
	if len(m.files) == 0 {
		return tea.Quit
	}

	// Launch up to m.parallel operations concurrently.
	n := min(m.parallel, len(m.files))
	var cmds []tea.Cmd
	for i := range n {
		m.files[i].state = stateUploading
		if m.isRescan {
			cmds = append(cmds,
				rescanFileCmd(i, m.web, m.files[i].sha256, m.token),
				m.files[i].spinner.Tick,
			)
		} else {
			cmds = append(cmds,
				uploadFileCmd(i, m.web, m.files[i].filename, m.token),
				m.files[i].spinner.Tick,
			)
		}
	}
	return tea.Batch(cmds...)
}

func (m scanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		// Update all active spinners.
		for i := range m.files {
			if m.files[i].state == stateUploading || m.files[i].state == stateScanning {
				var cmd tea.Cmd
				m.files[i].spinner, cmd = m.files[i].spinner.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case fileUploadedMsg:
		i := msg.index
		if msg.err != nil {
			m.files[i].state = stateError
			m.files[i].err = msg.err
			return m, m.maybeQuitOrNext()
		}
		m.files[i].sha256 = msg.sha256
		m.files[i].state = stateScanning
		cmds = append(cmds, pollStatusCmd(i, m.web, msg.sha256))

	case fileScanStatusMsg:
		i := msg.index
		if msg.err != nil {
			m.files[i].state = stateError
			m.files[i].err = msg.err
			return m, m.maybeQuitOrNext()
		}
		if msg.status == statusCompleted {
			cmds = append(cmds, fetchResultCmd(i, m.web, m.files[i].sha256))
		} else {
			m.files[i].pollCount++
			if m.files[i].pollCount >= maxPollRetries {
				m.files[i].state = stateError
				m.files[i].err = fmt.Errorf("scan timed out after %d attempts", maxPollRetries)
				return m, m.maybeQuitOrNext()
			}
			// Poll again after a delay.
			cmds = append(cmds, delayedPollCmd(i, m.web, m.files[i].sha256))
		}

	case fileScanDoneMsg:
		i := msg.index
		if msg.err != nil {
			m.files[i].state = stateError
			m.files[i].err = msg.err
		} else {
			m.files[i].state = stateDone
			m.files[i].result = &msg.summary
		}
		cmd := m.maybeQuitOrNext()
		return m, cmd
	}

	return m, tea.Batch(cmds...)
}

// maybeQuitOrNext launches pending uploads up to the parallel limit, or quits if all done.
func (m *scanModel) maybeQuitOrNext() tea.Cmd {
	// Count in-flight files (uploading or scanning).
	inFlight := 0
	allDone := true
	for _, f := range m.files {
		switch f.state {
		case stateUploading, stateScanning:
			inFlight++
			allDone = false
		case statePending:
			allDone = false
		}
	}
	if allDone {
		m.done = true
		return tea.Quit
	}

	// Launch pending files up to the parallel limit.
	var cmds []tea.Cmd
	for i := range m.files {
		if inFlight >= m.parallel {
			break
		}
		if m.files[i].state == statePending {
			m.files[i].state = stateUploading
			if m.isRescan {
				cmds = append(cmds,
					rescanFileCmd(i, m.web, m.files[i].sha256, m.token),
					m.files[i].spinner.Tick,
				)
			} else {
				cmds = append(cmds,
					uploadFileCmd(i, m.web, m.files[i].filename, m.token),
					m.files[i].spinner.Tick,
				)
			}
			inFlight++
		}
	}
	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}
	return nil
}

// --- Styles ---

var (
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))  // green
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))  // red
	styleWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))  // yellow
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // dim gray
	styleLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // blue
)

func (m scanModel) View() string {
	var s string
	for _, f := range m.files {
		name := filepath.Base(f.filename)
		switch f.state {
		case statePending:
			s += styleDim.Render("  "+name) + "\n"

		case stateUploading:
			label := " Uploading  "
			if m.isRescan {
				label = " Rescanning "
			}
			s += f.spinner.View() + styleLabel.Render(label) + name + " ...\n"

		case stateScanning:
			sha := truncSha(f.sha256)
			s += f.spinner.View() + styleLabel.Render(" Scanning   ") + name + " " + styleDim.Render(sha) + "\n"

		case stateDone:
			sha := truncSha(f.sha256)
			line := styleSuccess.Render("✓") + " " + name + "  " + styleDim.Render(sha)
			if f.result != nil {
				fmtStr := f.result.FileFormat
				if f.result.FileExtension != "" {
					fmtStr += "/" + f.result.FileExtension
				}
				line += "  " + fmtStr
				line += "  " + renderClassification(f.result.Classification)
				if f.result.MultiAV != nil {
					line += "  " + fmt.Sprintf("%d/%d engines",
						f.result.MultiAV.Positives, f.result.MultiAV.EnginesCount)
				}
			}
			s += line + "\n"

		case stateError:
			s += styleError.Render("✗") + " " + name + "  " + styleError.Render(f.err.Error()) + "\n"
		}
	}
	return s
}

func renderClassification(c string) string {
	switch c {
	case "malicious":
		return styleError.Render(c)
	case "clean":
		return styleSuccess.Render(c)
	case "unknown":
		return styleWarning.Render(c)
	default:
		return styleDim.Render(c)
	}
}

func truncSha(sha string) string {
	if len(sha) >= 12 {
		return sha[:12]
	}
	return sha
}
