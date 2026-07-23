package parser

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

const lineBreak = "\n"

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// htmlToMD converts HTML to a best-effort Markdown representation.
// Shared by all ATS parsers that process HTML content.
func htmlToMD(raw string) string {
	replacers := []struct{ from, to string }{
		{"<p>", lineBreak},
		{"</p>", lineBreak},
		{"<br>", lineBreak},
		{"<br/>", lineBreak},
		{"<br />", lineBreak},
		{"<li>", lineBreak + "- "},
		{"</li>", ""},
		{"<ul>", ""},
		{"</ul>", lineBreak},
		{"<ol>", ""},
		{"</ol>", lineBreak},
		{"<strong>", "**"},
		{"</strong>", "**"},
		{"<em>", "_"},
		{"</em>", "_"},
		{"<h1>", "# "},
		{"</h1>", lineBreak},
		{"<h2>", "## "},
		{"</h2>", lineBreak},
		{"<h3>", "### "},
		{"</h3>", lineBreak},
	}

	out := raw
	for _, pair := range replacers {
		out = strings.ReplaceAll(out, pair.from, pair.to)
	}

	out = htmlTagRe.ReplaceAllString(out, "")
	out = html.UnescapeString(out)

	return strings.TrimSpace(out)
}
