package chrome

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/chromedp/chromedp"
)

func TestCookie(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Test")
	}))
	defer ts.Close()

	chrome := Headless()
	defer chrome.Close()

	u, _ := url.Parse(ts.URL)
	chrome.SetCookies(u, []*http.Cookie{{Name: "test", Value: "value"}})
	if err := chrome.Run(chromedp.Navigate(ts.URL)); err != nil {
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
