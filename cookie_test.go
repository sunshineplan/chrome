package chrome

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func testHeadless() *Chrome {
	return Headless().SetDebuggerOutput(os.Stderr).NoSandbox()
}

func TestCookie(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Test")
	}))
	defer ts.Close()

	c := testHeadless()
	defer c.Close()

	ctx, cancel := context.WithTimeout(c, 10*time.Second)
	defer cancel()

	u, _ := url.Parse(ts.URL)
	SetCookies(ctx, u, []*http.Cookie{{Name: "test", Value: "value"}})
	if err := chromedp.Run(ctx, chromedp.Navigate(ts.URL)); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, i := range Cookies(ctx, u) {
		if i.Name == "test" && i.Value == "value" {
			found = true
			break
		}
	}
	if !found {
		t.Error("want found, got not found")
	}
}
