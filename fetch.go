package chrome

import (
	"context"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// EnableFetch enables request/response interception using the Fetch API.
// The provided function is called for each paused request and should return true to allow it or false to block it.
func EnableFetch(ctx context.Context, fn func(*fetch.EventRequestPaused) bool) error {
	paused := chromedp.Events(ctx, fetch.RequestPaused)
	go func() {
		for ev, err := range paused {
			if err != nil {
				return
			}
			if fn(&ev) {
				chromedp.Call(
					ctx,
					fetch.ContinueRequest,
					fetch.ContinueRequestParams{RequestID: ev.RequestID},
				)
			} else {
				chromedp.Call(
					ctx,
					fetch.FailRequest,
					fetch.FailRequestParams{
						RequestID:   ev.RequestID,
						ErrorReason: network.ErrorReasonBlockedByClient,
					},
				)
			}
		}
	}()
	_, err := chromedp.Call(ctx, fetch.Enable, fetch.EnableParams{})
	return err
}

// EnableFetch enables request/response interception on this Chrome instance.
func (c *Chrome) EnableFetch(fn func(*fetch.EventRequestPaused) bool) error {
	return EnableFetch(c, fn)
}
