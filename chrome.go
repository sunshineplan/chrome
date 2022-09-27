package chrome

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

var UnsetWebDriver = addScriptToEvaluateOnNewDocument("Object.defineProperty(navigator,'webdriver',{get:()=>false})")

var _ http.CookieJar = &Chrome{}

type Chrome struct {
	url     string
	flags   []chromedp.ExecAllocatorOption
	ctxOpts []chromedp.ContextOption
	actions []chromedp.Action

	ctx context.Context
}

func New(url string) *Chrome { return &Chrome{url: url} }

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

func (c *Chrome) context(timeout time.Duration) (ctx context.Context, cancel context.CancelFunc, err error) {
	if c.ctx == nil || c.ctx.Err() != nil {
		if timeout > 0 {
			ctx, cancel = context.WithTimeout(context.Background(), timeout)
		} else {
			ctx, cancel = context.WithCancel(context.Background())
		}

		if c.url == "" {
			ctx, _ = chromedp.NewExecAllocator(ctx, append(chromedp.DefaultExecAllocatorOptions[:], c.flags...)...)
		} else {
			ctx, _ = chromedp.NewRemoteAllocator(ctx, c.url)
		}
		ctx, _ = chromedp.NewContext(ctx, c.ctxOpts...)

		c.ctx = ctx
	} else {
		if timeout > 0 {
			ctx, cancel = context.WithTimeout(c.ctx, timeout)
			ctx, _ = chromedp.NewContext(ctx, c.ctxOpts...)
		} else {
			ctx, cancel = chromedp.NewContext(c.ctx, c.ctxOpts...)
		}

	}

	if err = chromedp.Run(ctx, c.actions...); err != nil {
		cancel()
		return nil, nil, err
	}

	return
}

func (c *Chrome) Context() (context.Context, context.CancelFunc, error) {
	return c.context(0)
}

func (c *Chrome) ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc, error) {
	return c.context(timeout)
}

func (c *Chrome) SetCookies(u *url.URL, cookies []*http.Cookie) {
	SetCookies(c.ctx, u, cookies)
}

func (c *Chrome) Cookies(u *url.URL) []*http.Cookie {
	return Cookies(c.ctx, u)
}

func addScriptToEvaluateOnNewDocument(script string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) (err error) {
		_, err = page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
		return
	})
}
