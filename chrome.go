package chrome

import (
	"context"
	"fmt"
	"strconv"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

var UnsetWebDriver = addScriptToEvaluateOnNewDocument("Object.defineProperty(navigator,'webdriver',{get:()=>false})")

type Chrome struct {
	url     string
	flags   []chromedp.ExecAllocatorOption
	ctxOpts []chromedp.ContextOption
	actions []chromedp.Action
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

func (c *Chrome) NewContext(ctx context.Context) (context.Context, context.CancelFunc, error) {
	if c.url == "" {
		ctx, _ = chromedp.NewExecAllocator(ctx, append(chromedp.DefaultExecAllocatorOptions[:], c.flags...)...,
		)
	} else {
		ctx, _ = chromedp.NewRemoteAllocator(ctx, c.url)
	}

	ctx, cancel := chromedp.NewContext(ctx, c.ctxOpts...)

	if err := chromedp.Run(ctx, c.actions...); err != nil {
		cancel()
		return nil, nil, err
	}

	return ctx, cancel, nil
}

func (c *Chrome) Context() (context.Context, context.CancelFunc, error) {
	return c.NewContext(context.Background())
}

func addScriptToEvaluateOnNewDocument(script string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) (err error) {
		_, err = page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
		return
	})
}
