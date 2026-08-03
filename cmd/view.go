// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/saferwall/cli/internal/entity"
	"github.com/saferwall/cli/internal/webapi"
	"github.com/spf13/cobra"
)

// viewBehaviorID selects which behavior report (sandbox run) to display in
// the Dynamic Analysis section.
var viewBehaviorID string

var viewCmd = &cobra.Command{
	Use:   "view <sha256>",
	Short: "View scan results for a file by its SHA256 hash",
	Long:  `Fetches and displays the scan results summary for a file, including AV detections.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sha256 := strings.ToLower(args[0])

		webSvc := webapi.New(cfg.Credentials.URL)
		var file entity.File
		if err := webSvc.GetFile(sha256, &file); err != nil {
			return fmt.Errorf("failed to get file: %w", err)
		}

		printFileReport(file, webSvc)
		return printDynamicAnalysis(file, webSvc, viewBehaviorID)
	},
}

func init() {
	viewCmd.Flags().StringVarP(&viewBehaviorID, "behavior-id", "b", "",
		"behavior report ID (sandbox run) to display in the Dynamic Analysis section")
	rootCmd.AddCommand(viewCmd)
}

// Styles for the report output.
var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	keyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	detectStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	cleanStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	avNameStyle = lipgloss.NewStyle().Width(24)
)

func printFileReport(file entity.File, webSvc webapi.Service) {
	fmt.Println()
	fmt.Println(titleStyle.Render("File Report"))
	fmt.Println(strings.Repeat("─", 60))

	// File identification.
	fmt.Println(headerStyle.Render("Identification"))
	printKV("SHA256", file.SHA256)
	if name := submissionFilename(file.Submissions); name != "" {
		printKV("Filename", name)
	}
	if !file.IsArchive {
		printKV("MD5", file.MD5)
		printKV("SHA1", file.SHA1)
		if file.SSDeep != "" {
			printKV("SSDeep", file.SSDeep)
		}
	}
	printKV("Size", formatSize(file.Size))
	fmt.Println()

	// File properties.
	fmt.Println(headerStyle.Render("Properties"))
	if file.Magic != "" {
		printKV("Magic", file.Magic)
	}
	if file.Format != "" {
		fmtStr := file.Format
		if file.Extension != "" {
			fmtStr += " (." + file.Extension + ")"
		}
		printKV("Format", fmtStr)
	}
	if !file.IsArchive && len(file.Packer) > 0 {
		printKV("Packer", strings.Join(file.Packer, ", "))
	}
	if file.IsArchive {
		printKV("Archive", fmt.Sprintf("yes (%d files)", len(file.DerivedFiles)))
	}
	if file.Encrypted {
		printKV("Encrypted", "yes")
		if file.DecryptionSuccess != nil {
			if *file.DecryptionSuccess {
				printKV("Decryption", cleanStyle.Render("successful"))
				if file.SuccessfulPassword != "" {
					printKV("Password", file.SuccessfulPassword)
				}
			} else {
				printKV("Decryption", detectStyle.Render("failed"))
				if len(file.AttemptedPasswords) > 0 {
					printKV("Attempted", strings.Join(file.AttemptedPasswords, ", "))
				}
			}
		}
	}
	if file.ParentSHA256 != "" {
		printKV("Parent", file.ParentSHA256)
	}
	if file.FirstSeen != 0 {
		printKV("First Seen", formatTimestamp(file.FirstSeen))
	}
	if file.LastScanned != 0 {
		printKV("Last Scanned", formatTimestamp(file.LastScanned))
	}
	fmt.Println()

	if file.IsArchive {
		// Archives only scan their children, skip verdict and AV results.
		if len(file.DerivedFiles) > 0 {
			printArchiveChildren(file.DerivedFiles, webSvc)
		}
	} else {
		// Classification.
		fmt.Println(headerStyle.Render("Classification"))
		printKV("Verdict", renderClassification(file.Classification))
		fmt.Println()

		// MultiAV results.
		printMultiAVResults(file.MultiAV)
	}
}

// childSummary holds the minimal info we display per archive child.
type childSummary struct {
	sha256         string
	classification string
	format         string
	size           int64
	positives      int
	enginesCount   int
	err            error
}

func fetchChildSummary(sha256 string, webSvc webapi.Service) childSummary {
	var file entity.File
	if err := webSvc.GetFile(sha256, &file); err != nil {
		return childSummary{sha256: sha256, err: err}
	}
	cs := childSummary{
		sha256:         sha256,
		classification: file.Classification,
		format:         file.Format,
		size:           file.Size,
	}
	if file.Extension != "" {
		cs.format += "/" + file.Extension
	}
	if lastScan, ok := file.MultiAV["last_scan"].(map[string]any); ok {
		if stats, ok := lastScan["stats"].(map[string]any); ok {
			if v, ok := stats["positives"].(float64); ok {
				cs.positives = int(v)
			}
			if v, ok := stats["engines_count"].(float64); ok {
				cs.enginesCount = int(v)
			}
		}
	}
	return cs
}

func printArchiveChildren(derivedFiles []entity.DerivedFile, webSvc webapi.Service) {
	fmt.Println(headerStyle.Render(fmt.Sprintf("Archive Contents (%d files)", len(derivedFiles))))
	fmt.Println()

	// Table header.
	nameCol := lipgloss.NewStyle().Width(28)
	fmtCol := lipgloss.NewStyle().Width(16)
	sizeCol := lipgloss.NewStyle().Width(10)
	avCol := lipgloss.NewStyle().Width(14)
	clsCol := lipgloss.NewStyle().Width(12)

	fmt.Printf("  %s  %s %s %s %s %s\n",
		styleDim.Render(fmt.Sprintf("%-64s", "SHA256")),
		styleDim.Render(nameCol.Render("NAME")),
		styleDim.Render(fmtCol.Render("FORMAT")),
		styleDim.Render(sizeCol.Render("SIZE")),
		styleDim.Render(avCol.Render("DETECTIONS")),
		styleDim.Render(clsCol.Render("VERDICT")),
	)
	fmt.Printf("  %s\n", styleDim.Render(strings.Repeat("─", 148)))

	for _, df := range derivedFiles {
		cs := fetchChildSummary(df.SHA256, webSvc)
		if cs.err != nil {
			fmt.Printf("  %s  %s\n",
				df.SHA256,
				styleError.Render("error: "+cs.err.Error()),
			)
			continue
		}

		detStr := fmt.Sprintf("%d/%d", cs.positives, cs.enginesCount)
		if cs.positives > 0 {
			detStr = detectStyle.Render(detStr)
		} else {
			detStr = cleanStyle.Render(detStr)
		}

		name := df.Name
		if len(name) > 28 {
			name = name[:25] + "..."
		}

		fmt.Printf("  %s  %s %s %s %s %s\n",
			cs.sha256,
			nameCol.Render(name),
			fmtCol.Render(cs.format),
			sizeCol.Render(formatSize(cs.size)),
			avCol.Render(detStr),
			clsCol.Render(renderClassification(cs.classification)),
		)
	}
	fmt.Println()
}

func printMultiAVResults(multiav map[string]any) {
	if multiav == nil {
		fmt.Println(headerStyle.Render("Antivirus Results"))
		fmt.Println("  No scan results available.")
		return
	}

	lastScan, ok := multiav["last_scan"].(map[string]any)
	if !ok {
		fmt.Println(headerStyle.Render("Antivirus Results"))
		fmt.Println("  No scan results available.")
		return
	}

	// Extract stats.
	var positives, enginesCount int
	if stats, ok := lastScan["stats"].(map[string]any); ok {
		if v, ok := stats["positives"].(float64); ok {
			positives = int(v)
		}
		if v, ok := stats["engines_count"].(float64); ok {
			enginesCount = int(v)
		}
	}

	// Summary line.
	fmt.Println(headerStyle.Render("Antivirus Results"))
	detectionStr := fmt.Sprintf("%d/%d engines detected this file", positives, enginesCount)
	if positives > 0 {
		fmt.Println("  " + detectStyle.Render(detectionStr))
	} else {
		fmt.Println("  " + cleanStyle.Render(detectionStr))
	}
	fmt.Println()

	// Collect detected engines only (engines live under last_scan.detections).
	type avResult struct {
		name   string
		output string
	}
	var detected []avResult
	var clean []avResult

	detections, _ := lastScan["detections"].(map[string]any)
	for key, val := range detections {
		engine, ok := val.(map[string]any)
		if !ok {
			continue
		}

		infected, _ := engine["infected"].(bool)
		output, _ := engine["output"].(string)
		if infected {
			detected = append(detected, avResult{name: key, output: output})
		} else {
			clean = append(clean, avResult{name: key})
		}
	}

	sort.Slice(detected, func(i, j int) bool { return detected[i].name < detected[j].name })
	sort.Slice(clean, func(i, j int) bool { return clean[i].name < clean[j].name })

	// Print detections.
	if len(detected) > 0 {
		for _, r := range detected {
			name := avNameStyle.Render(r.name)
			fmt.Printf("  %s %s\n", detectStyle.Render("●")+" "+name, detectStyle.Render(r.output))
		}
		fmt.Println()
	}

	// Print clean engines.
	if len(clean) > 0 {
		cleanNames := make([]string, len(clean))
		for i, r := range clean {
			cleanNames[i] = r.name
		}
		fmt.Printf("  %s %s\n", cleanStyle.Render("○"), styleDim.Render("No detection: "+strings.Join(cleanNames, ", ")))
		fmt.Println()
	}
}

func printKV(key, value string) {
	fmt.Printf("  %s %s\n", keyStyle.Render(fmt.Sprintf("%-14s", key+":")), value)
}

func formatSize(size int64) string {
	switch {
	case size >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(1<<30))
	case size >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func formatTimestamp(ts int64) string {
	t := time.Unix(ts, 0)
	return t.Format("2006-01-02 15:04:05 UTC")
}

// submissionFilename returns the first submission filename that does not look
// like a bare hash (MD5/SHA1/SHA256), or empty string if none is found.
func submissionFilename(submissions []entity.Submission) string {
	for _, s := range submissions {
		if s.Filename != "" && !looksLikeHash(s.Filename) {
			return s.Filename
		}
	}
	return ""
}

// selectBehaviorID picks the behavior run to show in detail: the requested ID
// wins regardless of status (so failed runs can be inspected), then the
// default report if completed, then the newest completed run. Returns an
// empty string when no run is selectable, and an error only when an
// explicitly requested ID does not exist on the file.
func selectBehaviorID(reports map[string]entity.BehaviorReportSummary, defaultID, requested string) (string, error) {
	if requested != "" {
		if _, ok := reports[requested]; ok {
			return requested, nil
		}
		return "", fmt.Errorf("behavior report %s not found for this file", requested)
	}

	if def, ok := reports[defaultID]; ok && def.Status == entity.BehaviorStatusCompleted {
		return defaultID, nil
	}

	bestID := ""
	var bestTS int64 = -1
	for id, r := range reports {
		if r.Status != entity.BehaviorStatusCompleted {
			continue
		}
		ts := behaviorRunTime(r)
		if ts > bestTS || (ts == bestTS && id > bestID) {
			bestTS, bestID = ts, id
		}
	}
	return bestID, nil
}

// behaviorRunTime returns the most relevant timestamp of a run for ordering.
func behaviorRunTime(r entity.BehaviorReportSummary) int64 {
	if r.FinishedAt != 0 {
		return r.FinishedAt
	}
	if r.StartedAt != 0 {
		return r.StartedAt
	}
	return r.QueuedAt
}

// behaviorDetail aggregates the results of the parallel behavior fetches.
type behaviorDetail struct {
	env          entity.BehaviorEnvironment
	envErr       error
	capabilities []entity.Capability
	processTree  []entity.Process
	counts       map[string]int // event type -> total; absent key means the count failed
}

// fetchBehaviorDetail fans out the behavior document and sys-event count
// requests concurrently. `env` is always present on a behavior document;
// `capabilities` and `proc_tree` are omitted when empty and must be fetched
// individually since a projection on a missing field fails the whole lookup —
// an error on those simply means there is nothing to display.
func fetchBehaviorDetail(webSvc webapi.Service, id string) behaviorDetail {
	d := behaviorDetail{counts: make(map[string]int)}

	eventTypes := []string{"network", "file", "registry"}
	countVals := make([]int, len(eventTypes))
	countErrs := make([]error, len(eventTypes))

	var wg sync.WaitGroup
	wg.Add(3 + len(eventTypes))
	go func() {
		defer wg.Done()
		var doc entity.Behavior
		if d.envErr = webSvc.GetBehaviorReport(id, []string{"env"}, &doc); d.envErr == nil {
			d.env = doc.Environment
		}
	}()
	go func() {
		defer wg.Done()
		var doc entity.Behavior
		if err := webSvc.GetBehaviorReport(id, []string{"capabilities"}, &doc); err == nil {
			d.capabilities = doc.Capabilities
		}
	}()
	go func() {
		defer wg.Done()
		var doc entity.Behavior
		if err := webSvc.GetBehaviorReport(id, []string{"proc_tree"}, &doc); err == nil {
			d.processTree = doc.ProcessTree
		}
	}()
	for i, t := range eventTypes {
		go func(i int, t string) {
			defer wg.Done()
			countVals[i], countErrs[i] = webSvc.CountSysEvents(id, t)
		}(i, t)
	}
	wg.Wait()

	for i, t := range eventTypes {
		if countErrs[i] == nil {
			d.counts[t] = countVals[i]
		}
	}
	return d
}

// printDynamicAnalysis renders the Dynamic Analysis section of the report.
func printDynamicAnalysis(file entity.File, webSvc webapi.Service, requestedID string) error {
	selectedID, err := selectBehaviorID(file.BehaviorReports, file.DefaultBehaviorID, requestedID)
	if err != nil {
		return err
	}

	fmt.Println(headerStyle.Render("Dynamic Analysis"))
	if len(file.BehaviorReports) == 0 {
		fmt.Println("  " + styleDim.Render("No dynamic analysis available."))
		fmt.Println()
		return nil
	}

	if selectedID != "" {
		sum := file.BehaviorReports[selectedID]
		detail := fetchBehaviorDetail(webSvc, selectedID)
		printBehaviorRunDetail(selectedID, sum, detail)
	}

	printOtherBehaviorRuns(file.BehaviorReports, selectedID)
	return nil
}

// renderBehaviorStatus colors a behavior run status.
func renderBehaviorStatus(status string) string {
	switch status {
	case entity.BehaviorStatusCompleted:
		return cleanStyle.Render(status)
	case entity.BehaviorStatusPartial, entity.BehaviorStatusFailed:
		return detectStyle.Render(status)
	default:
		return styleDim.Render(status)
	}
}

// renderPaddedBehaviorStatus is renderBehaviorStatus with the raw text
// left-padded to a fixed width before styling, for column alignment.
func renderPaddedBehaviorStatus(status string) string {
	padded := fmt.Sprintf("%-10s", status)
	switch status {
	case entity.BehaviorStatusCompleted:
		return cleanStyle.Render(padded)
	case entity.BehaviorStatusPartial, entity.BehaviorStatusFailed:
		return detectStyle.Render(padded)
	default:
		return styleDim.Render(padded)
	}
}

// printBehaviorRunDetail renders the selected run: environment, evidence,
// capabilities, process tree and activity counts.
func printBehaviorRunDetail(id string, sum entity.BehaviorReportSummary, d behaviorDetail) {
	printKV("Report ID", id)
	printKV("Status", renderBehaviorStatus(sum.Status))
	if sum.ScanConfig.OS != "" {
		printKV("OS", sum.ScanConfig.OS)
	}
	if sum.ScanConfig.Country != "" {
		printKV("Country", sum.ScanConfig.Country)
	}
	if d.envErr == nil && d.env.SandboxVersion != "" {
		sandbox := "v" + d.env.SandboxVersion
		if d.env.GuestHealthy {
			sandbox += styleDim.Render(" (guest healthy)")
		}
		printKV("Sandbox", sandbox)
	}
	if sum.StartedAt != 0 {
		printKV("Started", formatTimestamp(sum.StartedAt))
	}
	if sum.FinishedAt != 0 {
		printKV("Finished", formatTimestamp(sum.FinishedAt))
	}
	if sum.FinishedAt > sum.StartedAt && sum.StartedAt != 0 {
		printKV("Duration", (time.Duration(sum.FinishedAt-sum.StartedAt) * time.Second).String())
	}
	if sum.AttemptCount > 1 {
		printKV("Attempts", fmt.Sprintf("%d", sum.AttemptCount))
	}

	if sum.Failure != nil {
		failure := sum.Failure.Class
		if sum.Failure.Stage != "" {
			failure += " at stage " + sum.Failure.Stage
		}
		if sum.Failure.Message != "" {
			failure += ": " + sum.Failure.Message
		}
		printKV("Failure", detectStyle.Render(failure))
	}

	ev := sum.Evidence
	printKV("Evidence", fmt.Sprintf(
		"malware YARA: %d · high behavior: %d · high other: %d · medium: %d · rules: %d · detected artifacts: %d",
		ev.MalwareYARA, ev.HighBehavior, ev.HighOther, ev.Medium, ev.TotalRules, ev.DetectedArtifacts))
	printKV("Activity", renderActivityCounts(sum, d.counts))
	fmt.Println()

	if d.envErr != nil {
		fmt.Println("  " + styleError.Render("warning: could not fetch behavior report details: "+d.envErr.Error()))
		fmt.Println()
	}
	printCapabilities(d.capabilities)
	printProcessTree(d.processTree)
}

// renderActivityCounts builds the one-line activity summary. Counts whose
// fetch failed render as n/a.
func renderActivityCounts(sum entity.BehaviorReportSummary, counts map[string]int) string {
	count := func(eventType string) string {
		if n, ok := counts[eventType]; ok {
			return fmt.Sprintf("%d", n)
		}
		return "n/a"
	}
	return fmt.Sprintf("network events: %s · file events: %s · registry events: %s · artifacts: %d · screenshots: %d",
		count("network"), count("file"), count("registry"), sum.ArtifactCount, sum.ScreenshotsCount)
}

// severityRank orders capability groups from most to least severe.
func severityRank(severity string) int {
	switch severity {
	case "high":
		return 0
	case "suspicious":
		return 1
	case "informative":
		return 2
	default:
		return 3
	}
}

// renderSeverity colors a capability severity label, left-padded to a fixed
// width before styling so ANSI codes don't break column alignment.
func renderSeverity(severity string) string {
	padded := fmt.Sprintf("%-12s", severity)
	switch severity {
	case "high":
		return detectStyle.Render(padded)
	case "suspicious":
		return warnStyle.Render(padded)
	default:
		return styleDim.Render(padded)
	}
}

// printCapabilities renders detected capabilities grouped by severity,
// deduplicated by (severity, description).
func printCapabilities(caps []entity.Capability) {
	if len(caps) == 0 {
		return
	}

	seen := make(map[string]bool)
	var unique []entity.Capability
	for _, c := range caps {
		key := c.Severity + "\x00" + c.Description
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, c)
	}

	sort.SliceStable(unique, func(i, j int) bool {
		ri, rj := severityRank(unique[i].Severity), severityRank(unique[j].Severity)
		if ri != rj {
			return ri < rj
		}
		return unique[i].Description < unique[j].Description
	})

	fmt.Println(headerStyle.Render(fmt.Sprintf("Capabilities (%d)", len(unique))))
	for _, c := range unique {
		bullet := "●"
		switch severityRank(c.Severity) {
		case 0:
			bullet = detectStyle.Render(bullet)
		case 1:
			bullet = warnStyle.Render(bullet)
		default:
			bullet = styleDim.Render(bullet)
		}
		origin := c.Category
		if c.Module != "" {
			origin += "/" + c.Module
		}
		fmt.Printf("  %s %s %s %s\n",
			bullet, renderSeverity(c.Severity), c.Description, styleDim.Render("("+origin+")"))
	}
	fmt.Println()
}

// procNode is one node of the nested process tree.
type procNode struct {
	proc     entity.Process
	children []*procNode
}

// buildProcTree nests a flat process list by parent PID. A process whose
// parent is absent from the set (or empty) becomes a root. Children are
// sorted by PID; cycles and self-parenting cannot loop because each process
// is attached exactly once.
func buildProcTree(procs []entity.Process) []*procNode {
	nodes := make(map[string]*procNode, len(procs))
	order := make([]*procNode, 0, len(procs))
	for _, p := range procs {
		if _, ok := nodes[p.PID]; ok {
			continue
		}
		n := &procNode{proc: p}
		nodes[p.PID] = n
		order = append(order, n)
	}

	var roots []*procNode
	for _, n := range order {
		parent, ok := nodes[n.proc.ParentPID]
		if !ok || parent == n {
			roots = append(roots, n)
			continue
		}
		parent.children = append(parent.children, n)
	}

	sortNodes := func(ns []*procNode) {
		sort.Slice(ns, func(i, j int) bool { return ns[i].proc.PID < ns[j].proc.PID })
	}
	sortNodes(roots)
	for _, n := range order {
		sortNodes(n.children)
	}

	// Detached cycles (e.g. A→B→A with no root ancestor) never reach a root.
	// Promote each still-unreachable node to a root and sever its parent
	// edge, which breaks the cycle so tree walks terminate.
	reachable := make(map[*procNode]bool)
	var mark func(n *procNode)
	mark = func(n *procNode) {
		if reachable[n] {
			return
		}
		reachable[n] = true
		for _, c := range n.children {
			mark(c)
		}
	}
	for _, r := range roots {
		mark(r)
	}
	for _, n := range order {
		if reachable[n] {
			continue
		}
		if parent, ok := nodes[n.proc.ParentPID]; ok {
			for i, c := range parent.children {
				if c == n {
					parent.children = append(parent.children[:i], parent.children[i+1:]...)
					break
				}
			}
		}
		roots = append(roots, n)
		mark(n)
	}
	return roots
}

// renderProcTree flattens the nested tree into display lines, indented two
// spaces per depth level.
func renderProcTree(roots []*procNode) []string {
	var lines []string
	var walk func(n *procNode, depth int)
	walk = func(n *procNode, depth int) {
		name := n.proc.ProcessName
		if name == "" {
			name = n.proc.ImagePath
		}
		line := fmt.Sprintf("%s└─ %s %s", strings.Repeat("  ", depth), name, styleDim.Render("("+n.proc.PID+")"))
		if det := n.proc.Detection; det != "" && det != "clean" {
			line += " " + detectStyle.Render("["+det+"]")
		}
		lines = append(lines, line)
		for _, c := range n.children {
			walk(c, depth+1)
		}
	}
	for _, r := range roots {
		walk(r, 0)
	}
	return lines
}

// printProcessTree renders the nested process tree.
func printProcessTree(procs []entity.Process) {
	if len(procs) == 0 {
		return
	}
	fmt.Println(headerStyle.Render(fmt.Sprintf("Process Tree (%d)", len(procs))))
	for _, line := range renderProcTree(buildProcTree(procs)) {
		fmt.Println("  " + line)
	}
	fmt.Println()
}

// printOtherBehaviorRuns lists the remaining runs as one-liners, newest first.
func printOtherBehaviorRuns(reports map[string]entity.BehaviorReportSummary, selectedID string) {
	type run struct {
		id  string
		sum entity.BehaviorReportSummary
	}
	var others []run
	for id, sum := range reports {
		if id == selectedID {
			continue
		}
		others = append(others, run{id: id, sum: sum})
	}
	if len(others) == 0 {
		return
	}

	sort.Slice(others, func(i, j int) bool {
		ti, tj := behaviorRunTime(others[i].sum), behaviorRunTime(others[j].sum)
		if ti != tj {
			return ti > tj
		}
		return others[i].id > others[j].id
	})

	fmt.Println(headerStyle.Render(fmt.Sprintf("Other Runs (%d)", len(others))))
	for _, r := range others {
		line := fmt.Sprintf("  %s %s", r.id, renderPaddedBehaviorStatus(r.sum.Status))
		if r.sum.ScanConfig.OS != "" {
			line += " " + r.sum.ScanConfig.OS
		}
		if ts := behaviorRunTime(r.sum); ts != 0 {
			line += " " + styleDim.Render(formatTimestamp(ts))
		}
		line += fmt.Sprintf(" rules:%d", r.sum.Evidence.TotalRules)
		if r.sum.Failure != nil {
			line += " " + styleDim.Render(r.sum.Failure.Class)
		}
		fmt.Println(line)
	}
	fmt.Println("  " + styleDim.Render("Use --behavior-id <id> to view a specific run."))
	fmt.Println()
}
