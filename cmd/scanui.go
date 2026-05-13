// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	filename   string
	sha256     string
	size       int64 // file size in bytes (set from upload response for archives)
	state      fileState
	spinner    spinner.Model
	result     *scanSummary
	err        error
	pollCount  int
	isArchive  bool // true for ZIP containers with multiple files
	childCount int  // number of extracted files
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
	index    int
	sha256   string
	size     int64
	err      error
	isArchive bool
	children []entity.DerivedFile
}

type fileScanStatusMsg struct {
	index  int
	status int
	err    error
}

type fileScanDoneMsg struct {
	index    int
	summary  scanSummary
	isArchive bool
	children []entity.DerivedFile
	err      error
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
			file, err := web.Scan(filename, token, osFlag, enableDetonationFlag, timeoutFlag)
			if err != nil {
				return fileUploadedMsg{index: index, err: fmt.Errorf("upload: %w", err)}
			}
			// Don't use is_archive from the upload response: the file hasn't been
			// processed yet, so the field is always false at this point. Archive
			// detection happens in fetchResultCmd once the parent scan completes.
			return fileUploadedMsg{
				index:  index,
				sha256: file.SHA256,
				size:   file.Size,
			}
		} else if forceRescanFlag {
			// Fetch the existing file to check if it's an archive.
			var file entity.File
			if err := web.GetFile(sha256, &file); err != nil {
				return fileUploadedMsg{index: index, err: fmt.Errorf("get file: %w", err)}
			}

			if file.IsArchive && len(file.DerivedFiles) > 0 {
				// Archive: rescan each child, not the container itself.
				for _, df := range file.DerivedFiles {
					if err := web.Rescan(df.SHA256, token, osFlag, enableDetonationFlag, timeoutFlag); err != nil {
						return fileUploadedMsg{index: index, err: fmt.Errorf("rescan child %s: %w", df.SHA256[:12], err)}
					}
				}
				return fileUploadedMsg{
					index:     index,
					sha256:    sha256,
					size:      file.Size,
					isArchive: true,
					children:  file.DerivedFiles,
				}
			}

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
		return fileScanDoneMsg{
			index:     index,
			summary:   buildScanSummary(file),
			isArchive: file.IsArchive,
			children:  file.DerivedFiles,
		}
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
		// Check if the hash is an archive container.
		var file entity.File
		if err := web.GetFile(sha256, &file); err != nil {
			return fileUploadedMsg{index: index, err: fmt.Errorf("get file: %w", err)}
		}

		if file.IsArchive && len(file.DerivedFiles) > 0 {
			for _, df := range file.DerivedFiles {
				if err := web.Rescan(df.SHA256, token, osFlag, enableDetonationFlag, timeoutFlag); err != nil {
					return fileUploadedMsg{index: index, err: fmt.Errorf("rescan child %s: %w", df.SHA256[:12], err)}
				}
			}
			return fileUploadedMsg{
				index:     index,
				sha256:    sha256,
				size:      file.Size,
				isArchive: true,
				children:  file.DerivedFiles,
			}
		}

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

		if msg.isArchive && len(msg.children) > 0 {
			// Archive container: poll parent for completion and track children.
			m.files[i].state = stateScanning
			m.files[i].isArchive = true
			m.files[i].childCount = len(msg.children)
			m.files[i].size = msg.size
			cmds = append(cmds, pollStatusCmd(i, m.web, msg.sha256))

			archiveName := filepath.Base(m.files[i].filename)
			for _, df := range msg.children {
				s := spinner.New()
				s.Spinner = spinner.Dot
				m.files = append(m.files, fileRow{
					filename: archiveName + "/" + childDisplayName(df),
					sha256:   df.SHA256,
					state:    stateScanning,
					spinner:  s,
				})
				childIdx := len(m.files) - 1
				cmds = append(cmds,
					pollStatusCmd(childIdx, m.web, df.SHA256),
					m.files[childIdx].spinner.Tick,
				)
			}
		} else {
			m.files[i].state = stateScanning
			cmds = append(cmds, pollStatusCmd(i, m.web, msg.sha256))
		}

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
			// Late archive detection: for new uploads is_archive is false at
			// upload time and only becomes true once the backend processes the
			// file. The rescan/forceRescan paths pre-populate isArchive via
			// fileUploadedMsg, so skip them here to avoid adding duplicate rows.
			if msg.isArchive && !m.files[i].isArchive && len(msg.children) > 0 {
				m.files[i].isArchive = true
				m.files[i].childCount = len(msg.children)
				archiveName := filepath.Base(m.files[i].filename)
				for _, df := range msg.children {
					s := spinner.New()
					s.Spinner = spinner.Dot
					m.files = append(m.files, fileRow{
						filename: archiveName + "/" + childDisplayName(df),
						sha256:   df.SHA256,
						state:    stateScanning,
						spinner:  s,
					})
					childIdx := len(m.files) - 1
					cmds = append(cmds,
						pollStatusCmd(childIdx, m.web, df.SHA256),
						m.files[childIdx].spinner.Tick,
					)
				}
			}
		}
		cmds = append(cmds, m.maybeQuitOrNext())
		return m, tea.Batch(cmds...)
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

const maxVisibleCompleted = 10 // show only the last N completed/errored files

func (m scanModel) View() string {
	var s string

	// Count states for the progress header.
	var pending, inFlight, completed, errored int
	for _, f := range m.files {
		switch f.state {
		case statePending:
			pending++
		case stateUploading, stateScanning:
			inFlight++
		case stateDone:
			completed++
		case stateError:
			errored++
		}
	}
	total := len(m.files)
	finished := completed + errored

	// Progress header.
	s += styleLabel.Render(fmt.Sprintf("Progress: %d/%d", finished, total))
	if pending > 0 {
		s += styleDim.Render(fmt.Sprintf("  (%d pending)", pending))
	}
	if errored > 0 {
		s += "  " + styleError.Render(fmt.Sprintf("%d failed", errored))
	}
	s += "\n\n"

	// Show in-flight files (uploading/scanning) — these always get a spinner line.
	for _, f := range m.files {
		name := filepath.Base(f.filename)
		switch f.state {
		case stateUploading:
			label := " Uploading  "
			if m.isRescan {
				label = " Rescanning "
			}
			s += f.spinner.View() + styleLabel.Render(label) + displayName(name) + " ...\n"
		case stateScanning:
			s += f.spinner.View() + styleLabel.Render(" Scanning   ") + displayName(name) + " " + styleDim.Render(f.sha256) + "\n"
		}
	}

	// Collect completed/errored rows, show only the last N.
	type doneRow struct {
		line string
	}
	var doneRows []doneRow
	for _, f := range m.files {
		name := filepath.Base(f.filename)
		switch f.state {
		case stateDone:
			line := styleSuccess.Render("✓") + " " + displayName(name) + "  " + styleDim.Render(f.sha256)
			if f.isArchive {
				size := f.size
				if f.result != nil {
					size = f.result.Size
				}
				line += "  " + styleDim.Render(formatSize(size))
				line += "  " + styleLabel.Render(fmt.Sprintf("archive (%d files)", f.childCount))
			} else if f.result != nil {
				line += "  " + styleDim.Render(formatSize(f.result.Size))
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
			if f.result != nil && f.result.Encrypted {
				line += renderEncryptionStatus(f.result)
			}
			doneRows = append(doneRows, doneRow{line})
		case stateError:
			line := styleError.Render("✗") + " " + displayName(name) + "  " + styleError.Render(f.err.Error())
			doneRows = append(doneRows, doneRow{line})
		}
	}

	if len(doneRows) > maxVisibleCompleted && !m.done {
		hidden := len(doneRows) - maxVisibleCompleted
		s += styleDim.Render(fmt.Sprintf("  ... %d more completed above ...", hidden)) + "\n"
		doneRows = doneRows[hidden:]
	}
	for _, r := range doneRows {
		s += r.line + "\n"
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

// looksLikeHash returns true if s is a hex string of a common hash length
// (MD5=32, SHA1=40, SHA256=64).
func looksLikeHash(s string) bool {
	if len(s) != 32 && len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// displayName returns the name as-is, unless it looks like a hash, in which
// case it is truncated to the first 12 characters.
func displayName(name string) string {
	if looksLikeHash(name) {
		return truncSha(name)
	}
	return name
}

// childDisplayName returns the archive entry name when available, falling back
// to the truncated SHA256 so the row always has a meaningful label.
func childDisplayName(df entity.DerivedFile) string {
	if df.Name != "" {
		return df.Name
	}
	return truncSha(df.SHA256)
}

func renderEncryptionStatus(s *scanSummary) string {
	if s.DecryptionSuccess == nil {
		return "  " + styleWarning.Render("encrypted")
	}
	if *s.DecryptionSuccess {
		out := "  " + styleSuccess.Render("decrypted")
		if s.SuccessfulPassword != "" {
			out += " " + styleDim.Render("(pwd: "+s.SuccessfulPassword+")")
		}
		return out
	}
	out := "  " + styleError.Render("decryption failed")
	if len(s.AttemptedPasswords) > 0 {
		out += " " + styleDim.Render("(tried: "+strings.Join(s.AttemptedPasswords, ", ")+")")
	}
	return out
}

