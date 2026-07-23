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
	ErrJobrightHTTPStatus = errors.New("jobright: unexpected HTTP status")
	errBadJobrightURL     = errors.New("unrecognised jobright URL")
)

// FetchJobright parses a Jobright.ai job listing URL.
func FetchJobright(ctx context.Context, rawURL string) (*ParsedJob, error) {
	lower := strings.ToLower(rawURL)
	if !strings.Contains(lower, "jobright.ai") {
		return nil, fmt.Errorf("%s: %w", rawURL, errBadJobrightURL)
	}

	return FetchJobrightFromURL(ctx, rawURL, rawURL)
}

// FetchJobrightFromURL fetches from fetchURL and records sourceURL as origin (injectable for tests).
func FetchJobrightFromURL(ctx context.Context, fetchURL, sourceURL string) (*ParsedJob, error) {
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
		return nil, fmt.Errorf("status %d: %w", resp.StatusCode, ErrJobrightHTTPStatus)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	title, company := jobrightTitleCompany(doc)

	bodyMD := jobrightBody(doc)

	return &ParsedJob{
		Title:     title,
		Company:   company,
		BodyMD:    bodyMD,
		Source:    string(ATSJobright),
		SourceURL: sourceURL,
	}, nil
}

// jobrightTitleCompany extracts title and company from meta tags and page content.
func jobrightTitleCompany(doc *goquery.Document) (title, company string) {
	ogTitle, _ := doc.Find(`meta[property="og:title"]`).Attr("content")
	ogTitle = strings.TrimSpace(ogTitle)

	// Strip " | Jobright" suffix
	for _, suffix := range []string{" | Jobright", " - Jobright", " | jobright.ai"} {
		if idx := strings.LastIndex(ogTitle, suffix); idx > 0 {
			ogTitle = ogTitle[:idx]
		}
	}

	// "Role - Company" or "Role at Company"
	for _, sep := range []string{" at ", " - "} {
		if idx := strings.LastIndex(ogTitle, sep); idx > 0 {
			title = strings.TrimSpace(ogTitle[:idx])
			company = strings.TrimSpace(ogTitle[idx+len(sep):])

			return title, company
		}
	}

	if ogTitle != "" {
		title = ogTitle
	} else {
		title = strings.TrimSpace(doc.Find("h1").First().Text())
	}

	// Try known Jobright company selectors
	company = strings.TrimSpace(doc.Find(".company-name, p.company-name, [data-testid='company-name']").First().Text())

	return title, company
}

// jobrightBody extracts description text from the page.
func jobrightBody(doc *goquery.Document) string {
	// Try the job-description container first
	descText := strings.TrimSpace(doc.Find("#job-description, .job-description, [data-testid='job-description']").Text())
	if descText != "" {
		return descText
	}

	// Fall back to paragraph/list text
	var parts []string

	doc.Find("p,li").Each(func(_ int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		if text != "" {
			parts = append(parts, text)
		}
	})

	return strings.Join(parts, "\n")
}
