package chrome

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type (
	URLHasPrefix string
	URLHasSuffix string
	URLContains  string
	URLEqual     string
)

type Event struct {
	ID       network.RequestID
	URL      string
	Method   string
	Response *network.Response
	Bytes    []byte
}

func (e *Event) Header() http.Header {
	header := make(http.Header)
	if resp := e.Response; resp != nil {
		for k, v := range resp.Headers {
			header.Set(k, fmt.Sprint(v))
		}
	}
	return header
}

func ListenEvent(ctx context.Context, url any, method string, download bool) <-chan *Event {
	var m sync.Map
	done := make(chan *Event, 1)
	chromedp.ListenTarget(ctx, func(v any) {
		switch ev := v.(type) {
		case *network.EventRequestWillBeSent:
			var b bool
			if url == nil || url == "" {
				b = true
			} else {
				switch url := url.(type) {
				case string:
					b = strings.HasPrefix(ev.Request.URL, url)
				case URLHasPrefix:
					b = strings.HasPrefix(ev.Request.URL, string(url))
				case URLHasSuffix:
					b = strings.HasSuffix(ev.Request.URL, string(url))
				case URLContains:
					b = strings.Contains(ev.Request.URL, string(url))
				case URLEqual:
					b = ev.Request.URL == string(url)
				case *regexp.Regexp:
					b = url.MatchString(ev.Request.URL)
				}
			}
			if b && (method == "" || strings.EqualFold(method, ev.Request.Method)) {
				m.Store(ev.RequestID, &Event{ev.RequestID, ev.Request.URL, ev.Request.Method, ev.RedirectResponse, nil})
			}
		case *network.EventResponseReceived:
			if v, ok := m.Load(ev.RequestID); ok {
				v.(*Event).Response = ev.Response
			}
		case *network.EventLoadingFinished:
			if v, ok := m.Load(ev.RequestID); ok {
				go func() { done <- v.(*Event) }()
			}
		}
	})

	c := make(chan *Event, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(c)
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

func ListenScriptEvent(
	ctx context.Context, script string, url any, method, variable string, download bool) (string, <-chan *Event, error) {
	expression := fmt.Sprintf(script, "v")
	if strings.HasSuffix(expression, "%!(EXTRA string=v)") {
		expression = script
	} else {
		if regexp.MustCompile(`%!.?\(.+?v?\)v?$`).MatchString(expression) {
			return "", nil, fmt.Errorf("format error: %s", expression)
		}

		if variable == "" {
			b := make([]byte, 8)
			rand.Read(b)
			variable = "chrome" + hex.EncodeToString(b)
		}
		if err := chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintln("let", variable), nil)); err != nil {
			return "", nil, err
		}
		expression = fmt.Sprintf(script, variable)
	}

	c := ListenEvent(ctx, url, method, download)
	if err := chromedp.Run(ctx, chromedp.Evaluate(expression, nil)); err != nil {
		return "", nil, err
	}

	return variable, c, nil
}

func ListenScript(ctx context.Context, script string, url any, method, variable string, result any) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	variable, c, err := ListenScriptEvent(ctx, script, url, method, variable, false)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c:
		return chromedp.Run(ctx, chromedp.Evaluate(variable, &result))
	}
}
