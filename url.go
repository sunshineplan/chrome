package chrome

import (
	"regexp"
	"strings"
)

// URLHasPrefix matches URLs that have the specified prefix.
type URLHasPrefix string

// URLHasSuffix matches URLs that have the specified suffix.
type URLHasSuffix string

// URLContains matches URLs that contain the specified string.
type URLContains string

// URLEqual matches URLs that are exactly equal to the specified string.
type URLEqual string

// URLBase matches URLs that start with the specified base URL (ignoring query parameters).
type URLBase string

// Match checks if a URL starts with the specified prefix.
func (u URLHasPrefix) Match(s string) bool {
	return strings.HasPrefix(s, string(u))
}

// Match checks if a URL ends with the specified suffix.
func (u URLHasSuffix) Match(s string) bool {
	return strings.HasSuffix(s, string(u))
}

// Match checks if a URL contains the specified string.
func (u URLContains) Match(s string) bool {
	return strings.Contains(s, string(u))
}

// Match checks if a URL is exactly equal to the specified string.
func (u URLEqual) Match(s string) bool {
	return s == string(u)
}

// Match checks if a URL matches the base URL (ignoring query parameters).
func (u URLBase) Match(s string) bool {
	str := string(u)
	if i := strings.Index(str, "?"); i > 0 {
		str = str[:i]
	}
	return strings.HasPrefix(s, str)
}

// match checks if a URL matches the provided pattern.
// Supports nil/empty (matches all), strings (prefix match), regexes, and custom matchers.
func match(url string, value any) bool {
	if value == nil || value == "" {
		return true
	} else {
		switch v := value.(type) {
		case string:
			return strings.HasPrefix(url, v)
		case *regexp.Regexp:
			return v.MatchString(url)
		case interface{ Match(string) bool }:
			return v.Match(url)
		default:
			panic("unsupported url type")
		}
	}
}
