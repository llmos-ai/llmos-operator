package utils

import (
	"net/url"
	"strings"
	"unicode"

	"github.com/llmos-ai/llmos-controller/pkg/settings"
)

func GetLocalLLMUrl() (*url.URL, error) {
	localLLMUrl := settings.LocalLLMServerURL.Get()
	return url.Parse(localLLMUrl)
}

// ReplaceAndLower replaces underscores and colons with hyphens and converts the string to lowercase.
func ReplaceAndLower(s string) string {
	// Use a strings.Builder for efficient string concatenation
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '_', ':':
			sb.WriteRune('-')
		default:
			sb.WriteRune(unicode.ToLower(r))
		}
	}
	return sb.String()
}

func ArrayStringContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
