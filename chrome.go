package chrome

import (
	"context"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

var NotWebDriver = addScriptToEvaluateOnNewDocument("Object.defineProperty(navigator,'webdriver',{get:()=>false})")

type Chrome struct {
	flags   []chromedp.ExecAllocatorOption
	actions []chromedp.Action
}

func New() *Chrome {
	return new(Chrome)
}

func Headless(webdriver bool) *Chrome {
	if webdriver {
		return New()
	}
	return New().AddActions(NotWebDriver)
}

func Headful(webdriver bool) *Chrome {
	chrome := New().AddFlags(chromedp.Flag("headless", false))
	if webdriver {
		return chrome
	}
	return chrome.AddActions(NotWebDriver)
}

func (c *Chrome) AddFlags(flags ...chromedp.ExecAllocatorOption) *Chrome {
	c.flags = append(c.flags, flags...)
	return c
}

func (c *Chrome) AddActions(actions ...chromedp.Action) *Chrome {
	c.actions = append(c.actions, actions...)
	return c
}

func (c *Chrome) Context() (context.Context, context.CancelFunc, error) {
	ctx, cancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:], c.flags...)...,
	)
	ctx, cancel = chromedp.NewContext(ctx)

	if err := chromedp.Run(ctx, c.actions...); err != nil {
		cancel()
		return nil, nil, err
	}

	return ctx, cancel, nil
}

func addScriptToEvaluateOnNewDocument(script string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) (err error) {
		_, err = page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
		return
	})
}
