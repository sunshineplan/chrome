package chrome

import (
	"regexp"
	"strings"
)

type (
	URLHasPrefix string
	URLHasSuffix string
	URLContains  string
	URLEqual     string
	URLBase      string
)

func (u URLHasPrefix) Match(s string) bool {
	return strings.HasPrefix(s, string(u))
}

func (u URLHasSuffix) Match(s string) bool {
	return strings.HasSuffix(s, string(u))
}

func (u URLContains) Match(s string) bool {
	return strings.Contains(s, string(u))
}

func (u URLEqual) Match(s string) bool {
	return s == string(u)
}

func (u URLBase) Match(s string) bool {
	str := string(u)
	if i := strings.Index(str, "?"); i > 0 {
		str = str[:i]
	}
	return strings.HasPrefix(s, str)
}

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
