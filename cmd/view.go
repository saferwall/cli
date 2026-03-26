// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/saferwall/cli/internal/entity"
	"github.com/saferwall/cli/internal/webapi"
	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view <sha256>",
	Short: "View scan results for a file by its SHA256 hash",
	Long:  `Fetches and displays the scan results summary for a file, including AV detections.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sha256 := strings.ToLower(args[0])

		webSvc := webapi.New(cfg.Credentials.URL)
		token, err := webSvc.Login(cfg.Credentials.Username, cfg.Credentials.Password)
		if err != nil {
			log.Fatalf("failed to login: %v", err)
		}
		_ = token

		var file entity.File
		if err := webSvc.GetFile(sha256, &file); err != nil {
			log.Fatalf("failed to get file: %v", err)
		}

		printFileReport(file)
	},
}

func init() {
	rootCmd.AddCommand(viewCmd)
}

// Styles for the report output.
var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	keyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	detectStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	cleanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	avNameStyle  = lipgloss.NewStyle().Width(24)
)

func printFileReport(file entity.File) {
	fmt.Println()
	fmt.Println(titleStyle.Render("File Report"))
	fmt.Println(strings.Repeat("─", 60))

	// File identification.
	fmt.Println(headerStyle.Render("Identification"))
	printKV("SHA256", file.SHA256)
	printKV("MD5", file.MD5)
	printKV("SHA1", file.SHA1)
	if file.SSDeep != "" {
		printKV("SSDeep", file.SSDeep)
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
	if len(file.Packer) > 0 {
		printKV("Packer", strings.Join(file.Packer, ", "))
	}
	if file.FirstSeen != 0 {
		printKV("First Seen", formatTimestamp(file.FirstSeen))
	}
	if file.LastScanned != 0 {
		printKV("Last Scanned", formatTimestamp(file.LastScanned))
	}
	fmt.Println()

	// Classification.
	fmt.Println(headerStyle.Render("Classification"))
	printKV("Verdict", renderClassification(file.Classification))
	fmt.Println()

	// MultiAV results.
	printMultiAVResults(file.MultiAV)
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
		return fmt.Sprintf("%.2f GB (%d bytes)", float64(size)/float64(1<<30), size)
	case size >= 1<<20:
		return fmt.Sprintf("%.2f MB (%d bytes)", float64(size)/float64(1<<20), size)
	case size >= 1<<10:
		return fmt.Sprintf("%.2f KB (%d bytes)", float64(size)/float64(1<<10), size)
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}

func formatTimestamp(ts int64) string {
	t := time.Unix(ts, 0)
	return t.Format("2006-01-02 15:04:05 UTC")
}
