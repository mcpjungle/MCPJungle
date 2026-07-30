package cmd

import "strings"

const (
	toolDescriptionSummaryLimit = 80
	toolDescriptionEllipsis     = "..."
)

func summarizeToolDescription(description string) string {
	summary := strings.Join(strings.Fields(description), " ")
	runes := []rune(summary)
	if len(runes) <= toolDescriptionSummaryLimit {
		return summary
	}

	textLimit := toolDescriptionSummaryLimit - len(toolDescriptionEllipsis)
	return strings.TrimSpace(string(runes[:textLimit])) + toolDescriptionEllipsis
}
