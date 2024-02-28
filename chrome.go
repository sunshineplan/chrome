package chrome

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

var (
	_ http.CookieJar  = &Chrome{}
	_ context.Context = &Chrome{}
)

type Chrome struct {
	url       string
	useragent string
	width     int
	height    int
	proxy     string

	flags   []chromedp.ExecAllocatorOption
	ctxOpts []chromedp.ContextOption
	actions []chromedp.Action

	mu sync.Mutex

	ctx    context.Context
	cancel chan struct{}
	done   chan struct{}
}

func getInfo() (userAgent string, width, height int) {
	c := New("").headless()
	defer c.Close()
	if err := c.Run(
		chromedp.Evaluate("navigator.userAgent", &userAgent),
		chromedp.Evaluate("window.screen.width", &width),
		chromedp.Evaluate("window.screen.height", &height),
	); err != nil {
		panic(err)
	}
	userAgent = strings.ReplaceAll(userAgent, "Headless", "")
	return
}

func New(url string) *Chrome { return &Chrome{url: url, cancel: make(chan struct{})} }

func (c *Chrome) headless() *Chrome {
	return c.AddFlags(
		chromedp.Flag("headless", "new"),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("mute-audio", true),
	)
}

func UserAgent() string {
	userAgent, _, _ := getInfo()
	return userAgent
}

func Headless() *Chrome {
	userAgent, width, height := getInfo()
	return New("").UserAgent(userAgent).WindowSize(width, height).headless()
}

func Headful() *Chrome {
	_, width, height := getInfo()
	return New("").WindowSize(width, height)
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

func (c *Chrome) UserAgent(useragent string) *Chrome {
	c.useragent = useragent
	return c
}

func (c *Chrome) WindowSize(width int, height int) *Chrome {
	c.width, c.height = width, height
	return c
}

func (c *Chrome) Proxy(proxy string) *Chrome {
	c.proxy = proxy
	return c
}

func (c *Chrome) AddFlags(flags ...chromedp.ExecAllocatorOption) *Chrome {
	c.flags = append(c.flags, flags...)
	return c
}

func (c *Chrome) AutoOpenDevtools() *Chrome {
	return c.AddFlags(chromedp.Flag("auto-open-devtools-for-tabs", true))
}

func (c *Chrome) DisableUserAgentClientHint() *Chrome {
	return c.AddFlags(chromedp.Flag("disable-features", "UserAgentClientHint"))
}

func (c *Chrome) DisableAutomationControlled() *Chrome {
	return c.AddFlags(chromedp.Flag("disable-blink-features", "AutomationControlled"))
}

func (c *Chrome) AddContextOptions(opts ...chromedp.ContextOption) *Chrome {
	c.ctxOpts = append(c.ctxOpts, opts...)
	return c
}

func (c *Chrome) AddActions(actions ...chromedp.Action) *Chrome {
	c.actions = append(c.actions, actions...)
	return c
}

var DefaultExecAllocatorOptions = [...]chromedp.ExecAllocatorOption{
	// https://github.com/puppeteer/puppeteer/blob/main/packages/puppeteer-core/src/node/ChromeLauncher.ts
	chromedp.Flag("allow-pre-commit-input", true),
	chromedp.Flag("disable-background-networking", true),
	chromedp.Flag("disable-background-timer-throttling", true),
	chromedp.Flag("disable-backgrounding-occluded-windows", true),
	chromedp.Flag("disable-breakpad", true),
	chromedp.Flag("disable-client-side-phishing-detection", true),
	chromedp.Flag("disable-component-extensions-with-background-pages", true),
	chromedp.Flag("disable-component-update", true),
	chromedp.Flag("disable-default-apps", true),
	chromedp.Flag("disable-dev-shm-usage", true),
	chromedp.Flag("disable-extensions", true),
	chromedp.Flag("disable-field-trial-config", true),
	chromedp.Flag("disable-hang-monitor", true),
	chromedp.Flag("disable-infobars", true),
	chromedp.Flag("disable-ipc-flooding-protection", true),
	chromedp.Flag("disable-popup-blocking", true),
	chromedp.Flag("disable-prompt-on-repost", true),
	chromedp.Flag("disable-renderer-backgrounding", true),
	chromedp.Flag("disable-search-engine-choice-screen", true),
	chromedp.Flag("disable-sync", true),
	chromedp.Flag("enable-automation", true),
	chromedp.Flag("export-tagged-pdf", true),
	// chromedp.Flag("force-color-profile", "srgb"),
	chromedp.Flag("metrics-recording-only", true),
	chromedp.Flag("no-first-run", true),
	chromedp.Flag("password-store", "basic"),
	chromedp.Flag("use-mock-keychain", true),
	chromedp.Flag("disable-features", "Translate,AcceptCHFrame,MediaRouter,OptimizationHints,ProcessPerSiteUpToMainFrameThreshold"),
	chromedp.Flag("enable-features", "NetworkServiceInProcess2"),
}

func (c *Chrome) context(ctx context.Context, reset bool) (context.Context, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var new bool
	if c.ctx == nil || (reset && c.Err() != nil) {
		var allocatorCancel, ctxCancel context.CancelFunc
		if c.url == "" {
			opts := DefaultExecAllocatorOptions[:]
			if c.useragent != "" {
				opts = append(opts, chromedp.UserAgent(c.useragent))
			}
			if c.width != 0 && c.height != 0 {
				opts = append(opts, chromedp.WindowSize(c.width, c.height))
			}
			if c.proxy != "" {
				opts = append(opts, chromedp.ProxyServer(c.proxy))
			}
			ctx, allocatorCancel = chromedp.NewExecAllocator(ctx, append(opts, c.flags...)...)
		} else {
			ctx, allocatorCancel = chromedp.NewRemoteAllocator(ctx, c.url)
		}
		c.ctx, ctxCancel = chromedp.NewContext(ctx, c.ctxOpts...)
		c.cancel, c.done = make(chan struct{}), make(chan struct{})
		new = true

		if err := c.Run(c.actions...); err != nil {
			ctxCancel()
			allocatorCancel()
			panic(err)
		}

		go func(fn1, fn2 func()) {
			select {
			case <-c.cancel:
			case <-ctx.Done():
			}
			fn1()
			fn2()
			c.cancel = nil
			close(c.done)
		}(ctxCancel, allocatorCancel)
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
			var timeoutCancel, ctxCancel context.CancelFunc
			ctx, timeoutCancel = context.WithTimeout(c, timeout)
			ctx, ctxCancel = chromedp.NewContext(ctx, c.ctxOpts...)
			cancel = func() { ctxCancel(); timeoutCancel() }
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

func (c *Chrome) Run(actions ...chromedp.Action) error {
	return chromedp.Run(c, actions...)
}

func (c *Chrome) WaitNewTarget(fn func(*target.Info) bool) <-chan target.ID {
	return chromedp.WaitNewTarget(c, fn)
}

func AddScriptToEvaluateOnNewDocument(script string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) (err error) {
		_, err = page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
		return
	})
}
