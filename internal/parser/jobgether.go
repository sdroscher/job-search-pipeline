package parser

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	ErrJobgetherHTTPStatus = errors.New("jobgether: unexpected HTTP status")
	errBadJobgetherURL     = errors.New("unrecognised jobgether URL")
)

// FetchJobgether parses a Jobgether job listing URL.
func FetchJobgether(ctx context.Context, rawURL string) (*ParsedJob, error) {
	lower := strings.ToLower(rawURL)
	if !strings.Contains(lower, "jobgether.com") {
		return nil, fmt.Errorf("%s: %w", rawURL, errBadJobgetherURL)
	}

	return FetchJobgetherFromURL(ctx, rawURL, rawURL)
}

// FetchJobgetherFromURL fetches from fetchURL and records sourceURL as origin (injectable for tests).
func FetchJobgetherFromURL(ctx context.Context, fetchURL, sourceURL string) (*ParsedJob, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (compatible; job-search-pipeline/1.0)")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %w", resp.StatusCode, ErrJobgetherHTTPStatus)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	title, company := jobgetherTitleCompany(doc)

	var bodyParts []string

	doc.Find("h1,h2,h3,p,li").Each(func(_ int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		if text != "" {
			bodyParts = append(bodyParts, text)
		}
	})

	return &ParsedJob{
		Title:     title,
		Company:   company,
		BodyMD:    strings.Join(bodyParts, "\n"),
		Source:    string(ATSJobgether),
		SourceURL: sourceURL,
	}, nil
}

// stripSuffix removes each suffix from s when found at a non-zero position.
func stripSuffix(s string, suffixes []string) string {
	for _, suffix := range suffixes {
		if idx := strings.LastIndex(s, suffix); idx > 0 {
			s = s[:idx]
		}
	}

	return s
}

// jobgetherTitleCompany extracts title from <h1> and company from og:title ("Role at Company").
// Falls back to og:title for title and .company-name selector for company when needed.
func jobgetherTitleCompany(doc *goquery.Document) (title, company string) {
	ogTitle, _ := doc.Find(`meta[property="og:title"]`).Attr("content")
	ogTitle = strings.TrimSpace(stripSuffix(ogTitle, []string{" | Jobgether", " - Jobgether"}))

	// Extract company from "Role at Company" pattern in og:title
	if idx := strings.LastIndex(ogTitle, " at "); idx > 0 {
		company = strings.TrimSpace(ogTitle[idx+4:])
	}

	// Prefer h1 for title (preserves full "Role at Company" wording)
	title = strings.TrimSpace(doc.Find("h1").First().Text())
	if title == "" {
		if idx := strings.LastIndex(ogTitle, " at "); idx > 0 {
			title = strings.TrimSpace(ogTitle[:idx])
		} else {
			title = ogTitle
		}
	}

	// Company fallback
	if company == "" {
		company = strings.TrimSpace(doc.Find(".company-name, [data-company], .employer-name").First().Text())
	}

	return title, company
}
