package chrome

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/chromedp/cdproto/domstorage"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

var UnsetWebDriver = AddScriptToEvaluateOnNewDocument("Object.defineProperty(navigator,'webdriver',{get:()=>undefined})")

var (
	_ http.CookieJar  = &Chrome{}
	_ context.Context = &Chrome{}
)

type Chrome struct {
	url     string
	flags   []chromedp.ExecAllocatorOption
	ctxOpts []chromedp.ContextOption
	actions []chromedp.Action

	mu sync.Mutex

	ctx    context.Context
	cancel chan struct{}
	done   chan struct{}
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
	if c.ctx == nil {
		c.context(context.Background(), false)
	}
	return c.ctx.Deadline()
}

func (c *Chrome) Done() <-chan struct{} {
	if c.ctx == nil {
		c.context(context.Background(), false)
	}
	return c.ctx.Done()
}

func (c *Chrome) Err() error {
	if c.ctx == nil {
		c.context(context.Background(), false)
	}
	return c.ctx.Err()
}

func (c *Chrome) Value(key any) any {
	if c.ctx == nil {
		c.context(context.Background(), false)
	}
	return c.ctx.Value(key)
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

func (c *Chrome) context(ctx context.Context, reset bool) (context.Context, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var new bool
	if c.ctx == nil || (reset && c.Err() != nil) {
		var cancel context.CancelFunc
		if c.url == "" {
			ctx, cancel = chromedp.NewExecAllocator(ctx, append(chromedp.DefaultExecAllocatorOptions[:], c.flags...)...)
		} else {
			ctx, cancel = chromedp.NewRemoteAllocator(ctx, c.url)
		}
		c.ctx, _ = chromedp.NewContext(ctx, c.ctxOpts...)
		c.cancel, c.done = make(chan struct{}), make(chan struct{})
		new = true

		if err := c.Run(c.actions...); err != nil {
			cancel()
			panic(err)
		}

		go func() {
			select {
			case <-c.cancel:
				cancel()
			case <-ctx.Done():
			}
			c.cancel = nil
			close(c.done)
		}()
	}

	return c, new
}

func (c *Chrome) newContext(timeout time.Duration) (ctx context.Context, cancel context.CancelFunc, err error) {
	var new bool
	if c.ctx == nil || c.Err() != nil {
		if timeout > 0 {
			ctx, cancel = context.WithTimeout(context.Background(), timeout)
		} else {
			ctx, cancel = context.WithCancel(context.Background())
		}
		ctx, new = c.context(ctx, true)
	}
	if !new {
		if timeout > 0 {
			ctx, cancel = context.WithTimeout(c, timeout)
			ctx, _ = chromedp.NewContext(ctx, c.ctxOpts...)
		} else {
			ctx, cancel = chromedp.NewContext(c, c.ctxOpts...)
		}

		if err = c.Run(c.actions...); err != nil {
			cancel()
			return nil, nil, err
		}
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
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		close(c.cancel)
		<-c.done
	}
}

func (c *Chrome) SetCookies(u *url.URL, cookies []*http.Cookie) {
	SetCookies(c, u, cookies)
}

func (c *Chrome) Cookies(u *url.URL) []*http.Cookie {
	return Cookies(c, u)
}

func (c *Chrome) SetStorageItem(storageID *domstorage.StorageID, key, value string) error {
	return SetStorageItem(c, storageID, key, value)
}

func (c *Chrome) StorageItems(storageID *domstorage.StorageID) ([]domstorage.Item, error) {
	return StorageItems(c, storageID)
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

func (c *Chrome) Run(actions ...chromedp.Action) error {
	return chromedp.Run(c, actions...)
}

func AddScriptToEvaluateOnNewDocument(script string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) (err error) {
		_, err = page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
		return
	})
}
