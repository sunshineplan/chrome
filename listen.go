package chrome

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type Event struct {
	ID    network.RequestID
	URL   string
	Bytes []byte
}

func ListenURL(ctx context.Context, url string, method string, download bool) <-chan Event {
	c, done := make(chan Event, 1), make(chan Event, 1)
	var m sync.Map
	chromedp.ListenTarget(ctx, func(v any) {
		switch ev := v.(type) {
		case *network.EventRequestWillBeSent:
			if strings.HasPrefix(ev.Request.URL, url) && method == ev.Request.Method {
				m.Store(ev.RequestID, ev.Request.URL)
			}
		case *network.EventLoadingFinished:
			if v, ok := m.Load(ev.RequestID); ok {
				done <- Event{ev.RequestID, v.(string), nil}
			}
		}
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-done:
				if download {
					if err := chromedp.Run(
						ctx,
						chromedp.ActionFunc(func(ctx context.Context) (err error) {
							e.Bytes, err = network.GetResponseBody(e.ID).Do(ctx)
							return
						}),
					); err != nil {
						log.Print(err)
					}
				}
				c <- e
			}
		}
	}()

	return c
}

func ListenScript(ctx context.Context, script, url, method, variable string, result any) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := ListenURL(ctx, url, method, false)
	if err := chromedp.Run(
		ctx,
		chromedp.Evaluate(fmt.Sprintln("let", variable), nil),
		chromedp.Evaluate(fmt.Sprintf(script, variable), nil),
	); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c:
		return chromedp.Run(ctx, chromedp.Evaluate(variable, &result))
	}
}
