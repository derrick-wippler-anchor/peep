package server_test

import (
	"strings"
	"testing"

	"github.com/derrick-wippler-anchor/host/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripFrontmatter(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid frontmatter is stripped leaving only body",
			input: "---\ntitle: Hello\nauthor: test\n---\nBody content here\n",
			want:  "Body content here\n",
		},
		{
			name:  "no frontmatter delimiter returns input unchanged",
			input: "# Just a heading\nSome text.\n",
			want:  "# Just a heading\nSome text.\n",
		},
		{
			name:  "opening delimiter but no closing returns input unchanged",
			input: "---\ntitle: Hello\nno closing delimiter",
			want:  "---\ntitle: Hello\nno closing delimiter",
		},
		{
			name:  "empty input returns empty output",
			input: "",
			want:  "",
		},
		{
			name:  "multiline frontmatter body is fully stripped",
			input: "---\ntitle: My Page\ndate: 2024-01-01\ntags:\n  - go\n  - web\n---\nActual content starts here.\nMore content.\n",
			want:  "Actual content starts here.\nMore content.\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := server.StripFrontmatter([]byte(tc.input))
			require.Equal(t, tc.want, string(got))
		})
	}
}

func TestRenderMarkdown(t *testing.T) {
	cases := []struct {
		name          string
		input         string
		wantContains  []string
		wantAbsent    []string
	}{
		{
			name:         "heading produces h1 element",
			input:        "# Hello World\n",
			wantContains: []string{"<h1>"},
		},
		{
			name:         "bold produces strong element",
			input:        "**bold text**\n",
			wantContains: []string{"<strong>"},
		},
		{
			// chroma produces inline styles on spans when highlighting
			name:         "fenced code block with language tag produces syntax-highlighted output",
			input:        "```go\npackage main\n\nfunc main() {}\n```\n",
			wantContains: []string{`<span style="`},
		},
		{
			name:         "GFM table produces table element",
			input:        "| Col1 | Col2 |\n|------|------|\n| A    | B    |\n",
			wantContains: []string{"<table>"},
		},
		{
			name:         "strikethrough produces del element",
			input:        "~~strikethrough~~\n",
			wantContains: []string{"<del>"},
		},
		{
			name:         "GFM task list produces checkbox input",
			input:        "- [ ] task item\n",
			wantContains: []string{`<input`, `type="checkbox"`},
		},
		{
			name:         "bare URL is rendered as anchor tag",
			input:        "https://example.com\n",
			wantContains: []string{"<a "},
		},
		{
			name:         "frontmatter content is absent from rendered output",
			input:        "---\ntitle: Secret Title\n---\n# Visible Heading\n",
			wantContains: []string{"<h1>"},
			wantAbsent:   []string{"Secret Title"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := server.RenderMarkdown([]byte(tc.input))
			require.NoError(t, err)

			body := string(got)
			for _, want := range tc.wantContains {
				assert.True(t, strings.Contains(body, want),
					"expected output to contain %q, got:\n%s", want, body)
			}
			for _, absent := range tc.wantAbsent {
				assert.False(t, strings.Contains(body, absent),
					"expected output NOT to contain %q, got:\n%s", absent, body)
			}
		})
	}
}
