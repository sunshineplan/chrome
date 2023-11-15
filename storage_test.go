package chrome

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chromedp/cdproto/domstorage"
	"github.com/chromedp/chromedp"
)

func TestStorage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Test")
	}))
	defer ts.Close()

	c := Headless()
	defer c.Close()

	ctx, cancel := context.WithTimeout(c, 10*time.Second)
	defer cancel()

	if err := chromedp.Run(ctx, chromedp.Navigate(ts.URL)); err != nil {
		t.Fatal(err)
	}

	storageID := &domstorage.StorageID{StorageKey: domstorage.SerializedStorageKey(ts.URL + "/")}
	if err := SetStorageItem(ctx, storageID, "test", "value"); err != nil {
		t.Fatal(err)
	}
	items, err := StorageItems(ctx, storageID)
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
