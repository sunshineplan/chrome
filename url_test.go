package chrome

import (
	"regexp"
	"testing"
)

func TestCompare(t *testing.T) {
	url := "https://github.com/sunshineplan/chrome?test=true"
	testcases := []any{
		"https://github.com",
		URLHasPrefix("https://github.com"),
		URLHasSuffix("test=true"),
		URLContains("chrome"),
		URLEqual("https://github.com/sunshineplan/chrome?test=true"),
		URLBase("https://github.com/sunshineplan/chrome?base=true"),
		regexp.MustCompile(`^https:\/\/github.com\/sunshineplan\/chrome\?test=true$`),
	}
	for _, tc := range testcases {
		if !match(url, tc) {
			t.Errorf("want true, got false. Value: %v", tc)
		}
	}
}
