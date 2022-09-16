package chrome

import (
	"os"
	"testing"

	"github.com/chromedp/cdproto/browser"
)

func TestDownload(t *testing.T) {
	ctx, cancel, err := Headless(true).Context()
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := SetDownload(ctx, wd); err != nil {
		t.Fatal(err)
	}

	res, err := Download(ctx, "https://github.com/sunshineplan/chrome/archive/refs/heads/main.zip", nil)
	if err != nil {
		t.Fatal(err)
	}

	if expect := "https://codeload.github.com/sunshineplan/chrome/zip/refs/heads/main"; res.Begin.URL != expect {
		t.Errorf("expected %q; got %q", expect, res.Begin.URL)
	}
	if expect := "chrome-main.zip"; res.Begin.SuggestedFilename != expect {
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
