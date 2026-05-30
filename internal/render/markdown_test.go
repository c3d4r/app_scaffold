package render

import (
	"strings"
	"testing"
)

func TestToHTML(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		contains []string // substrings that must appear
		absent   []string // substrings that must NOT appear
	}{
		{
			name:     "bold and list",
			src:      "**Bold** text\n\n- one\n- two",
			contains: []string{"<strong>Bold</strong>", "<ul>", "<li>one</li>"},
		},
		{
			name:     "fenced code highlighted",
			src:      "```go\nfmt.Println(\"hi\")\n```",
			contains: []string{"<pre", "<code", "style="}, // chroma inline styles
		},
		{
			name:     "inline math",
			src:      "Euler: $e^{i\\pi}+1=0$",
			contains: []string{`class="math inline"`, `\(`, `\)`},
		},
		{
			name:     "markdown image",
			src:      "![cat](https://example.com/cat.png)",
			contains: []string{"<img", `src="https://example.com/cat.png"`, `alt="cat"`},
		},
		{
			name:     "javascript link is sanitized",
			src:      "[click](javascript:alert(1))",
			absent:   []string{"javascript:alert"},
		},
		{
			name:     "raw html escaped not executed",
			src:      "<script>alert(1)</script> hi",
			absent:   []string{"<script>alert(1)</script>"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := string(ToHTML(c.src))
			for _, want := range c.contains {
				if !strings.Contains(got, want) {
					t.Errorf("expected output to contain %q\ngot: %s", want, got)
				}
			}
			for _, bad := range c.absent {
				if strings.Contains(got, bad) {
					t.Errorf("expected output to NOT contain %q\ngot: %s", bad, got)
				}
			}
		})
	}
}
