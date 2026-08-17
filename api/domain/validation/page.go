package validation

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

var pageSlugPattern = regexp.MustCompile(`^[a-z0-9/-]+$`)

func ValidatePageSlug(slug string) error {
	if !pageSlugPattern.MatchString(slug) || strings.HasPrefix(slug, "/") || strings.HasSuffix(slug, "/") || strings.Contains(slug, "//") {
		return errors.New("slug must use lowercase letters, digits, hyphens, and non-empty slash-separated segments")
	}
	return nil
}

func ValidatePageTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("title is required")
	}
	for _, r := range title {
		if unicode.IsControl(r) {
			return errors.New("title must not contain control characters")
		}
	}
	return nil
}

func ValidateTimelineEntry(entry string, required bool) error {
	if strings.ContainsAny(entry, "\r\n") {
		return errors.New("timeline entry must be one line")
	}
	if required && strings.TrimSpace(entry) == "" {
		return errors.New("timeline entry is required")
	}
	return nil
}

func ValidateCompiledTruth(content string) error {
	if strings.Contains(content, "<!-- timeline -->") {
		return errors.New("compiled truth must not contain the timeline delimiter")
	}
	return nil
}
