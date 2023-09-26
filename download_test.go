package chrome

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/browser"
)

func TestDownload(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="test.txt"`)
		http.ServeContent(w, r, "test.txt", time.Time{}, strings.NewReader("test download"))
	}))
	defer ts.Close()

	chrome := Headless()
	defer chrome.Close()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := chrome.SetDownload(wd); err != nil {
		t.Fatal(err)
	}

	res, err := chrome.Download(ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	if expect := ts.URL + "/"; res.Begin.URL != expect {
		t.Errorf("expected %q; got %q", expect, res.Begin.URL)
	}
	if expect := "test.txt"; res.Begin.SuggestedFilename != expect {
		t.Errorf("expected %q; got %q", expect, res.Begin.SuggestedFilename)
	}
	if res.Begin.GUID != res.Progress.GUID {
		t.Errorf("expected same GUID; got Begin: %s, Progress: %s", res.Begin.GUID, res.Progress.GUID)
	}
	if res.Progress.TotalBytes != res.Progress.ReceivedBytes {
		t.Errorf("expected same value; got Total: %g, Received: %g", res.Progress.TotalBytes, res.Progress.ReceivedBytes)
	}
	if expect := browser.DownloadProgressStateCompleted; res.Progress.State != expect {
		t.Errorf("expected %q; got %q", expect, res.Begin.SuggestedFilename)
	}

	info, err := os.Stat(res.GUID())
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(info.Name())

	if float64(info.Size()) != res.Progress.ReceivedBytes {
		t.Errorf("expected %g; got %d", res.Progress.ReceivedBytes, info.Size())
	}
}
