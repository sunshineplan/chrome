package chrome

import (
	"testing"

	"github.com/chromedp/cdproto/domstorage"
	"github.com/chromedp/chromedp"
)

func TestStorage(t *testing.T) {
	chrome := Headless()
	defer chrome.Close()

	if err := chrome.Run(chromedp.Navigate("https://github.com/sunshineplan/chrome")); err != nil {
		t.Fatal(err)
	}

	storageID := &domstorage.StorageID{SecurityOrigin: "https://github.com"}
	if err := chrome.SetStorageItem(storageID, "test", "value"); err != nil {
		t.Fatal(err)
	}
	items, err := chrome.StorageItems(storageID)
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, i := range items {
		if i[0] == "test" && i[1] == "value" {
			found = true
			break
		}
	}
	if !found {
		t.Error("want found, got not found")
	}
}
