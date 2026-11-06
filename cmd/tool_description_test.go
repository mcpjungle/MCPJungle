package cmd

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSummarizeToolDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        string
	}{
		{
			name:        "empty",
			description: "",
			want:        "",
		},
		{
			name:        "whitespace only",
			description: " \n\t\r ",
			want:        "",
		},
		{
			name:        "short description",
			description: "Find nearby places",
			want:        "Find nearby places",
		},
		{
			name:        "collapse whitespace",
			description: "Find nearby\n\tplaces   by name",
			want:        "Find nearby places by name",
		},
		{
			name:        "exact limit",
			description: strings.Repeat("a", toolDescriptionSummaryLimit),
			want:        strings.Repeat("a", toolDescriptionSummaryLimit),
		},
		{
			name:        "over limit",
			description: strings.Repeat("a", toolDescriptionSummaryLimit+1),
			want:        strings.Repeat("a", toolDescriptionSummaryLimit-len(toolDescriptionEllipsis)) + toolDescriptionEllipsis,
		},
		{
			name:        "unicode remains valid",
			description: strings.Repeat("界", toolDescriptionSummaryLimit+1),
			want:        strings.Repeat("界", toolDescriptionSummaryLimit-len(toolDescriptionEllipsis)) + toolDescriptionEllipsis,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeToolDescription(tt.description)
			if got != tt.want {
				t.Fatalf("summarizeToolDescription() = %q, want %q", got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("summarizeToolDescription() returned invalid UTF-8: %q", got)
			}
			if utf8.RuneCountInString(got) > toolDescriptionSummaryLimit {
				t.Fatalf("summary has %d runes, want at most %d", utf8.RuneCountInString(got), toolDescriptionSummaryLimit)
			}
		})
	}
}
