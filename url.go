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

func compare(url string, value any) (res bool) {
	if value == nil || value == "" {
		res = true
	} else {
		switch v := value.(type) {
		case string:
			res = strings.HasPrefix(url, v)
		case URLHasPrefix:
			res = strings.HasPrefix(url, string(v))
		case URLHasSuffix:
			res = strings.HasSuffix(url, string(v))
		case URLContains:
			res = strings.Contains(url, string(v))
		case URLEqual:
			res = url == string(v)
		case URLBase:
			str := string(v)
			if i := strings.Index(str, "?"); i > 0 {
				str = str[:i]
			}
			res = strings.HasPrefix(url, str)
		case *regexp.Regexp:
			res = v.MatchString(url)
		default:
			panic("unsupported url type")
		}
	}
	return
}
