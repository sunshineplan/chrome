package chrome

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

var DefaultChannelBufferCapacity = 8

type Event struct {
	Request  *network.EventRequestWillBeSent
	Response *network.EventResponseReceived
	Bytes    []byte
}

func (e *Event) Header() http.Header {
	header := make(http.Header)
	if resp := e.Response; resp != nil {
		for k, v := range resp.Response.Headers {
			header.Set(k, fmt.Sprint(v))
		}
	}
	return header
}

func (e *Event) Unmarshal(v any) error {
	return json.Unmarshal(e.Bytes, v)
}

func (e *Event) String() string {
	return string(e.Bytes)
}

func ListenEvent(ctx context.Context, url any, method string, download bool) <-chan *Event {
	c, ec := make(chan *Event, DefaultChannelBufferCapacity), make(chan *Event, DefaultChannelBufferCapacity)
	done := make(chan struct{})
	var wg sync.WaitGroup
	go func() {
		<-ctx.Done()
		wg.Wait()
		close(ec)
		<-done
		close(c)
	}()
	var m sync.Map
	chromedp.ListenTarget(ctx, func(v any) {
		switch ev := v.(type) {
		case *network.EventRequestWillBeSent:
			if match(ev.Request.URL, url) && (method == "" || strings.EqualFold(method, ev.Request.Method)) {
				m.Store(ev.RequestID, &Event{ev, nil, nil})
			}
		case *network.EventResponseReceived:
			if v, ok := m.Load(ev.RequestID); ok {
				v.(*Event).Response = ev
				if v.(*Event).Request.Request.Method == "HEAD" {
					m.Delete(ev.RequestID)
					wg.Add(1)
					go func() {
						defer wg.Done()
						select {
						case ec <- v.(*Event):
						case <-ctx.Done():
							return
						}
					}()
				}
			}
		case *network.EventLoadingFinished:
			if v, ok := m.LoadAndDelete(ev.RequestID); ok {
				wg.Add(1)
				go func() {
					defer wg.Done()
					select {
					case ec <- v.(*Event):
					case <-ctx.Done():
						return
					}
				}()
			}
		}
	})
	go func() {
		defer func() { close(done) }()
		for {
			select {
			case e, ok := <-ec:
				if !ok {
					return
				}
				if download {
					err := chromedp.Run(
						ctx,
						chromedp.ActionFunc(func(ctx context.Context) (err error) {
							e.Bytes, err = network.GetResponseBody(e.Response.RequestID).Do(ctx)
							return
						}),
					)
					if err != nil {
						slog.Debug(err.Error())
					}
				}
				select {
				case c <- e:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
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

func (c *Chrome) ListenEvent(url any, method string, download bool) <-chan *Event {
	return ListenEvent(c, url, method, download)
}

func (c *Chrome) ListenScriptEvent(script string, url any, method, variable string, download bool) (string, <-chan *Event, error) {
	return ListenScriptEvent(c, script, url, method, variable, download)
}

func (c *Chrome) ListenScript(script string, url any, method, variable string, result any) error {
	return ListenScript(c, script, url, method, variable, result)
}
