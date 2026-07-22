package parser

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var errHTMLStatus = errors.New("html scrape: unexpected status")

// ScrapeHTML fetches and scrapes an arbitrary HTML job posting page.
func ScrapeHTML(ctx context.Context, rawURL string) (*ParsedJob, error) {
	return ScrapeHTMLFromURL(ctx, rawURL, rawURL)
}

// ScrapeHTMLFromURL fetches from fetchURL and records sourceURL as the origin (used in tests).
func ScrapeHTMLFromURL(ctx context.Context, fetchURL, sourceURL string) (*ParsedJob, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	resp, err := http.DefaultClient.Do(httpReq)
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
