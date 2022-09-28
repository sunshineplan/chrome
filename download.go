package chrome

import (
	"context"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

type DownloadEvent struct {
	Begin    *browser.EventDownloadWillBegin
	Progress *browser.EventDownloadProgress
}

func (e *DownloadEvent) GUID() string {
	if e.Begin == nil {
		panic("pointer is nil")
	}
	return e.Begin.GUID
}

func ListenDownload(ctx context.Context, url any) <-chan *DownloadEvent {
	var m sync.Map
	c := make(chan *DownloadEvent, 1)
	chromedp.ListenTarget(ctx, func(v any) {
		switch ev := v.(type) {
		case *browser.EventDownloadWillBegin:
			if compare(ev.URL, url) {
				m.Store(ev.GUID, &DownloadEvent{ev, nil})
			}
		case *browser.EventDownloadProgress:
			if v, ok := m.Load(ev.GUID); ok && ev.State == browser.DownloadProgressStateCompleted {
				go func() {
					v := v.(*DownloadEvent)
					v.Progress = ev
					c <- v
				}()
			}
		}
	})

	return c
}

func SetDownload(ctx context.Context, path string) error {
	return chromedp.Run(ctx, browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
		WithDownloadPath(path).
		WithEventsEnabled(true))
}

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
