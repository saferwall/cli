// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/saferwall/cli/internal/webapi"
	"github.com/spf13/cobra"
)

var (
	searchPage    int
	searchPerPage int
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for files on the Saferwall platform",
	Long: `Search files using a query expression.

Examples:
  saferwall-cli search 'type=pe and positives>=10'
  saferwall-cli search 'fs>2026 and tag=upx' --per-page 50
  saferwall-cli search 'extension=sys and positives>=10' --page 2`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		webSvc := webapi.New(cfg.Credentials.URL)
		result, err := webSvc.SearchFiles(args[0], cfg.Credentials.APIKey, searchPage, searchPerPage)
		if err != nil {
			log.Fatalf("search failed: %v", err)
		}
		printSearchResults(result, searchPage, searchPerPage)
	},
}

func init() {
	searchCmd.Flags().IntVarP(&searchPage, "page", "p", 1, "Page number")
	searchCmd.Flags().IntVarP(&searchPerPage, "per-page", "n", 20, "Results per page")
}

func printSearchResults(result *webapi.SearchResult, page, perPage int) {
	fmt.Println()

	if result.TotalCount == 0 {
		fmt.Println("  No results found.")
		fmt.Println()
		return
	}

	// Summary line.
	start := (page-1)*perPage + 1
	end := start + len(result.Items) - 1
	fmt.Printf("  %s\n\n",
		headerStyle.Render(fmt.Sprintf("Showing %d-%d of %d results", start, end, result.TotalCount)),
	)

	// Column styles.
	nameCol := lipgloss.NewStyle().Width(24)
	typeCol := lipgloss.NewStyle().Width(16)
	sizeCol := lipgloss.NewStyle().Width(10)
	detCol := lipgloss.NewStyle().Width(12)
	dateCol := lipgloss.NewStyle().Width(12)
	clsCol := lipgloss.NewStyle().Width(12)

	// Header row.
	fmt.Printf("  %s  %s %s %s %s %s %s %s\n",
		styleDim.Render(fmt.Sprintf("%-64s", "SHA256")),
		styleDim.Render(nameCol.Render("NAME")),
		styleDim.Render(typeCol.Render("TYPE/EXT")),
		styleDim.Render(sizeCol.Render("SIZE")),
		styleDim.Render(detCol.Render("DETECTIONS")),
		styleDim.Render(dateCol.Render("FIRST SEEN")),
		styleDim.Render(dateCol.Render("LAST SCANNED")),
		styleDim.Render(clsCol.Render("VERDICT")),
	)
	fmt.Printf("  %s\n", styleDim.Render(strings.Repeat("─", 172)))

	for _, item := range result.Items {
		// Name: hide if it looks like a bare hash (the API echoes the ID as name).
		name := item.Name
		if name == "" || looksLikeHash(name) {
			name = "-"
		}
		if len(name) > 24 {
			name = name[:21] + "..."
		}

		// Type/extension column.
		typeStr := item.Format
		if item.Extension != "" {
			typeStr += "/" + item.Extension
		}
		if typeStr == "" {
			typeStr = "-"
		}
		if len(typeStr) > 16 {
			typeStr = typeStr[:13] + "..."
		}

		// AV detections from condensed multiav.hits/total.
		detStr := "-"
		if item.MultiAV.Total > 0 {
			raw := fmt.Sprintf("%d/%d", item.MultiAV.Hits, item.MultiAV.Total)
			if item.MultiAV.Hits > 0 {
				detStr = detectStyle.Render(raw)
			} else {
				detStr = cleanStyle.Render(raw)
			}
		}

		// Timestamps: date only.
		firstSeen := "-"
		if item.FirstSeen != 0 {
			firstSeen = time.Unix(item.FirstSeen, 0).UTC().Format("2006-01-02")
		}
		lastScanned := "-"
		if item.LastScanned != 0 {
			lastScanned = time.Unix(item.LastScanned, 0).UTC().Format("2006-01-02")
		}

		fmt.Printf("  %s  %s %s %s %s %s %s %s\n",
			item.ID,
			nameCol.Render(name),
			typeCol.Render(typeStr),
			sizeCol.Render(formatSize(item.Size)),
			detCol.Render(detStr),
			dateCol.Render(firstSeen),
			dateCol.Render(lastScanned),
			clsCol.Render(renderClassification(item.Classification)),
		)
	}

	fmt.Println()

	// Pagination hint.
	if result.PageCount > 1 {
		fmt.Printf("  %s\n\n",
			styleDim.Render(fmt.Sprintf("Page %d of %d — use --page to navigate", page, result.PageCount)),
		)
	}
}
