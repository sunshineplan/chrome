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
	c := make(chan *DownloadEvent, DefaultChannelBufferCapacity)
	go func() {
		<-ctx.Done()
		close(c)
	}()
	var m sync.Map
	chromedp.ListenTarget(ctx, func(v any) {
		switch ev := v.(type) {
		case *browser.EventDownloadWillBegin:
			if compare(ev.URL, url) {
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

func (c *Chrome) ListenDownload(url any) <-chan *DownloadEvent {
	return ListenDownload(c, url)
}

func (c *Chrome) SetDownload(path string) error {
	return SetDownload(c, path)
}

func (c *Chrome) Download(url string, match any) (*DownloadEvent, error) {
	return Download(c, url, match)
}
