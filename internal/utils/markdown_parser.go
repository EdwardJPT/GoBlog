package utils

import (
	"bytes"
	"html/template"
	"log/slog"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// Post holds the data for the template
type Post struct {
	Title   string
	Content template.HTML
}

// renderMarkdown converts raw strings to safe HTML
func RenderMarkdown(raw string) template.HTML {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,            // Tables, Task Lists, Strikethrough, Autolinks
			extension.Footnote,       // Footnotes support
			extension.DefinitionList, // Definition lists
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"), // 'dracula' looks great on #100340
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // For section links
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // Required to render the raw HTML tags in your test (e.g., <details>)
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(raw), &buf); err != nil {
		slog.Error("Failed converting markdown to a rendered content", "error", err)
		return ""
	}
	return template.HTML(buf.String())
}
