package chrome

import (
	"context"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

// DownloadEvent represents a download event with begin and progress information.
type DownloadEvent struct {
	Begin    *browser.EventDownloadWillBegin // Event fired when download is about to begin
	Progress *browser.EventDownloadProgress  // Event with download progress/completion info
}

// GUID returns the unique identifier for this download event.
func (e *DownloadEvent) GUID() string {
	if e.Begin == nil {
		panic("pointer is nil")
	}
	return e.Begin.GUID
}

// ListenDownload listens for download events matching the provided URL pattern.
// Returns a channel that receives completed DownloadEvents.
func ListenDownload(ctx context.Context, url any) <-chan *DownloadEvent {
	c := make(chan *DownloadEvent, DefaultChannelBufferCapacity)
	go func() {
		<-ctx.Done()
		close(c)
	}()
	var m sync.Map
	chromedp.ListenTarget(ctx, func(v any) {
		switch ev := v.(type) {
		case *browser.EventDownloadWillBegin:
			if match(ev.URL, url) {
				m.Store(ev.GUID, &DownloadEvent{ev, nil})
			}
		case *browser.EventDownloadProgress:
			if v, ok := m.Load(ev.GUID); ok && ev.State == browser.DownloadProgressStateCompleted {
				m.Delete(ev.GUID)
				v := v.(*DownloadEvent)
				v.Progress = ev
				go func() {
					select {
					case c <- v:
					case <-ctx.Done():
						return
					}
				}()
			}
		}
	})
	return c
}

// SetDownload configures the browser to save downloads to the specified path.
func SetDownload(ctx context.Context, path string) error {
	return chromedp.Run(ctx, browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
		WithDownloadPath(path).
		WithEventsEnabled(true))
}

// Download navigates to the given URL and waits for a download matching the provided pattern.
// It returns the completed DownloadEvent or an error.
func Download(ctx context.Context, url string, match any) (*DownloadEvent, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := ListenDownload(ctx, match)
	if err := chromedp.Run(ctx, chromedp.Navigate(url)); err != nil && !strings.Contains(err.Error(), "net::ERR_ABORTED") {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case e := <-c:
		return e, nil
	}
}

// ListenDownload listens for downloads from the browser instance.
func (c *Chrome) ListenDownload(url any) <-chan *DownloadEvent {
	return ListenDownload(c, url)
}

// SetDownload configures this Chrome instance to save downloads to the specified path.
func (c *Chrome) SetDownload(path string) error {
	return SetDownload(c, path)
}

// Download navigates to a URL and waits for a download from this Chrome instance.
func (c *Chrome) Download(url string, match any) (*DownloadEvent, error) {
	return Download(c, url, match)
}
