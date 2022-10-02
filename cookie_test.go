package chrome

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/chromedp/chromedp"
)

func TestCookie(t *testing.T) {
	chrome := Headless(true)
	defer chrome.Close()

	u := &url.URL{Scheme: "https", Host: "github.com"}
	chrome.SetCookies(u, []*http.Cookie{{Name: "test", Value: "value"}})
	if err := chrome.Run(chromedp.Navigate("https://github.com/sunshineplan/chrome")); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, i := range chrome.Cookies(u) {
		if i.Name == "test" && i.Value == "value" {
			found = true
			break
		}
	}
	if !found {
		t.Error("want found, got not found")
	}
}
