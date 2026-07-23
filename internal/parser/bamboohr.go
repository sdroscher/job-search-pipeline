package parser

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	bambooHRJobRe         = regexp.MustCompile(`([a-z0-9-]+)\.bamboohr\.com/(?:careers|jobs)/(\d+)`)
	errBadBambooHRURL     = errors.New("unrecognised bamboohr URL")
	errBambooHRHTTPStatus = errors.New("bamboohr page: unexpected status")
)

// FetchBambooHR parses a BambooHR job board URL by scraping the SSR'd HTML page.
func FetchBambooHR(ctx context.Context, rawURL string) (*ParsedJob, error) {
	if !bambooHRJobRe.MatchString(strings.ToLower(rawURL)) {
		return nil, fmt.Errorf("%s: %w", rawURL, errBadBambooHRURL)
	}

	return FetchBambooHRFromURL(ctx, rawURL, rawURL)
}

// FetchBambooHRFromURL fetches from fetchURL and records sourceURL as origin (injectable for tests).
func FetchBambooHRFromURL(ctx context.Context, fetchURL, sourceURL string) (*ParsedJob, error) {
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
		return nil, fmt.Errorf("status %d: %w", resp.StatusCode, errBambooHRHTTPStatus)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	title := bambooHRTitle(doc)

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
		Source:    string(ATSBambooHR),
		SourceURL: sourceURL,
	}, nil
}

// bambooHRTitle extracts the job title from og:title meta, falling back to <title>
// with BambooHR suffixes stripped.
func bambooHRTitle(doc *goquery.Document) string {
	if ogTitle, exists := doc.Find(`meta[property="og:title"]`).Attr("content"); exists && ogTitle != "" {
		return strings.TrimSpace(ogTitle)
	}

	raw := strings.TrimSpace(doc.Find("title").Text())

	for _, suffix := range []string{" | BambooHR", " - BambooHR"} {
		if idx := strings.LastIndex(raw, suffix); idx > 0 {
			raw = raw[:idx]
		}
	}

	return strings.TrimSpace(raw)
}
