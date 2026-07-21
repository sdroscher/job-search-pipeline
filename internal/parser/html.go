package parser

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var errHTMLStatus = errors.New("html scrape: unexpected status")

// ScrapeHTML fetches and scrapes an arbitrary HTML job posting page.
func ScrapeHTML(rawURL string) (*ParsedJob, error) {
	return ScrapeHTMLFromURL(rawURL, rawURL)
}

// ScrapeHTMLFromURL fetches from fetchURL and records sourceURL as the origin (used in tests).
func ScrapeHTMLFromURL(fetchURL, sourceURL string) (*ParsedJob, error) {
	resp, err := http.Get(fetchURL) //nolint:noctx,gosec
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %w", resp.StatusCode, errHTMLStatus)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	title, _ := doc.Find("title").First().Html()
	title = strings.TrimSpace(title)

	var bodyParts []string

	doc.Find("h1,h2,h3,p,li").Each(func(_ int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		if text != "" {
			bodyParts = append(bodyParts, text)
		}
	})

	return &ParsedJob{
		Title:     title,
		BodyMD:    strings.Join(bodyParts, "\n"),
		Source:    string(ATSHTML),
		SourceURL: sourceURL,
	}, nil
}
