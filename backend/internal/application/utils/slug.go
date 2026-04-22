package utils

import (
	"regexp"
	"strings"
	"unicode"
)

var slugRegexp = regexp.MustCompile(`[^a-z0-9]+`)

// GenerateSlug converts a name to a URL-safe slug: lowercase, spaces/punctuation → hyphens.
func GenerateSlug(name string) string {
	lower := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return '-'
	}, name)
	slug := slugRegexp.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "org"
	}
	return slug
}
