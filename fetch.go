package chrome

import (
	"context"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func EnableFetch(ctx context.Context, fn func(*fetch.EventRequestPaused) bool) error {
	chromedp.ListenTarget(ctx, func(v any) {
		switch ev := v.(type) {
		case *fetch.EventRequestPaused:
			go func() {
				ctx := cdp.WithExecutor(ctx, chromedp.FromContext(ctx).Target)
				if fn(ev) {
					fetch.ContinueRequest(ev.RequestID).Do(ctx)
				} else {
					fetch.FailRequest(ev.RequestID, network.ErrorReasonBlockedByClient).Do(ctx)
				}
			}()
		}
	})
	return chromedp.Run(ctx, fetch.Enable())
}

func (c *Chrome) EnableFetch(fn func(*fetch.EventRequestPaused) bool) error {
	return EnableFetch(c, fn)
}
