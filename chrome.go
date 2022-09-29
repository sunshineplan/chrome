package chrome

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

var UnsetWebDriver = addScriptToEvaluateOnNewDocument("Object.defineProperty(navigator,'webdriver',{get:()=>false})")

var (
	_ http.CookieJar  = &Chrome{}
	_ context.Context = &Chrome{}
)

type Chrome struct {
	url     string
	flags   []chromedp.ExecAllocatorOption
	ctxOpts []chromedp.ContextOption
	actions []chromedp.Action

	context.Context
	cancel chan struct{}
}

func New(url string) *Chrome { return &Chrome{url: url, cancel: make(chan struct{})} }

func Headless(webdriver bool) *Chrome {
	if webdriver {
		return New("")
	}
	return New("").AddActions(UnsetWebDriver)
}

func Headful(webdriver bool) *Chrome {
	chrome := New("").AddFlags(chromedp.Flag("headless", false))
	if webdriver {
		return chrome
	}
	return chrome.AddActions(UnsetWebDriver)
}

func Remote(url string) *Chrome {
	if url == "" {
		panic("empty url")
	}
	return New(url)
}

func Local(port int) *Chrome {
	if port <= 0 || port > 65535 {
		panic("invalid port number: " + strconv.Itoa(port))
	}
	return Remote(fmt.Sprintf("ws://localhost:%d", port))
}

func (c *Chrome) Deadline() (deadline time.Time, ok bool) {
	if c.Context == nil {
		c.context(context.Background())
	}
	return c.Context.Deadline()
}

func (c *Chrome) Done() <-chan struct{} {
	if c.Context == nil {
		c.context(context.Background())
	}
	return c.Context.Done()
}

func (c *Chrome) Err() error {
	if c.Context == nil {
		c.context(context.Background())
	}
	return c.Context.Err()
}

func (c *Chrome) Value(key any) any {
	if c.Context == nil {
		c.context(context.Background())
	}
	return c.Context.Value(key)
}

func (c *Chrome) AddFlags(flags ...chromedp.ExecAllocatorOption) *Chrome {
	c.flags = append(c.flags, flags...)
	return c
}

func (c *Chrome) AddContextOptions(opts ...chromedp.ContextOption) *Chrome {
	c.ctxOpts = append(c.ctxOpts, opts...)
	return c
}

func (c *Chrome) AddActions(actions ...chromedp.Action) *Chrome {
	c.actions = append(c.actions, actions...)
	return c
}

func (c *Chrome) context(ctx context.Context) context.Context {
	if c.url == "" {
		ctx, _ = chromedp.NewExecAllocator(ctx, append(chromedp.DefaultExecAllocatorOptions[:], c.flags...)...)
	} else {
		ctx, _ = chromedp.NewRemoteAllocator(ctx, c.url)
	}

	var cancel context.CancelFunc
	c.Context, cancel = chromedp.NewContext(ctx, c.ctxOpts...)

	go func() {
		<-c.cancel
		cancel()
	}()

	return c
}

func (c *Chrome) newContext(timeout time.Duration) (ctx context.Context, cancel context.CancelFunc, err error) {
	if c.Context == nil || c.Err() != nil {
		if timeout > 0 {
			ctx, cancel = context.WithTimeout(context.Background(), timeout)
		} else {
			ctx, cancel = context.WithCancel(context.Background())
		}
		ctx = c.context(ctx)
	} else {
		if timeout > 0 {
			ctx, cancel = context.WithTimeout(c, timeout)
			ctx, _ = chromedp.NewContext(ctx, c.ctxOpts...)
		} else {
			ctx, cancel = chromedp.NewContext(c, c.ctxOpts...)
		}
	}

	if err = chromedp.Run(ctx, c.actions...); err != nil {
		cancel()
		return nil, nil, err
	}

	return
}

func (c *Chrome) NewContext() (context.Context, context.CancelFunc, error) {
	return c.newContext(0)
}

func (c *Chrome) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc, error) {
	return c.newContext(timeout)
}

func (c *Chrome) Close() {
	if c.cancel != nil {
		close(c.cancel)
	}
}

func (c *Chrome) SetCookies(u *url.URL, cookies []*http.Cookie) {
	SetCookies(c, u, cookies)
}

func (c *Chrome) Cookies(u *url.URL) []*http.Cookie {
	return Cookies(c, u)
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

func (c *Chrome) ListenEvent(url any, method string, download bool) <-chan *Event {
	return ListenEvent(c, url, method, download)
}

func (c *Chrome) ListenScriptEvent(script string, url any, method, variable string, download bool) (string, <-chan *Event, error) {
	return ListenScriptEvent(c, script, url, method, variable, download)
}

func (c *Chrome) ListenScript(script string, url any, method, variable string, result any) error {
	return ListenScript(c, script, url, method, variable, result)
}

func (c *Chrome) EnableFetch(fn func(*fetch.EventRequestPaused) bool) error {
	return EnableFetch(c, fn)
}

func addScriptToEvaluateOnNewDocument(script string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) (err error) {
		_, err = page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
		return
	})
}
