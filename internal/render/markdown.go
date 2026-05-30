// Package render converts assistant message text (markdown) into safe HTML for
// display. It is the single rendering path: both the live poll response and a
// full page reload render message content through ToHTML, so output is
// identical regardless of how the message was produced.
package render

import (
	"bytes"
	"html/template"
	"sync"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	once   sync.Once
	md     goldmark.Markdown
	policy *bluemonday.Policy
)

func setup() {
	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			mathjax.MathJax, // $..$ -> <span class="math inline">\(..\)</span> (KaTeX renders client-side)
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				// inline styles (no separate stylesheet to ship); sanitizer allows them below
				highlighting.WithFormatOptions(chromahtml.WithClasses(false)),
			),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(), // single newline -> <br>, matches chat expectations
			// NOTE: no html.WithUnsafe() — raw HTML in the source stays escaped.
		),
	)

	// Base policy handles standard tags and, crucially, URL-scheme safety on
	// links/images (strips javascript:, etc.). Extended to permit the
	// attributes emitted by chroma (highlighting) and mathjax.
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").OnElements("span", "code", "pre", "div", "p")
	p.AllowAttrs("style").OnElements("span", "pre", "code")
	p.AllowStyles("color", "background-color", "font-weight", "font-style", "text-decoration").Globally()
	policy = p
}

// ToHTML renders markdown source to sanitized HTML safe for direct embedding.
func ToHTML(src string) template.HTML {
	once.Do(setup)

	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		// Fall back to escaped plain text rather than dropping the message.
		return template.HTML(template.HTMLEscapeString(src))
	}
	return template.HTML(policy.SanitizeBytes(buf.Bytes()))
}
