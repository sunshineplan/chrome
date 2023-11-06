package chrome

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

var (
	_ http.CookieJar  = &Chrome{}
	_ context.Context = &Chrome{}
)

type Chrome struct {
	url       string
	useragent string
	proxy     string

	flags   []chromedp.ExecAllocatorOption
	ctxOpts []chromedp.ContextOption
	actions []chromedp.Action

	mu sync.Mutex

	ctx    context.Context
	cancel chan struct{}
	done   chan struct{}
}

func New(url string) *Chrome { return &Chrome{url: url, cancel: make(chan struct{})} }

func Headless() *Chrome {
	c := New("")
	defer c.Close()
	var userAgent string
	if err := c.Run(chromedp.Evaluate("navigator.userAgent", &userAgent)); err != nil {
		panic(err)
	}
	re := regexp.MustCompile(`HeadlessChrome/(\d+)\.\d+.\d+.\d+`)
	return New("").
		AddFlags(chromedp.Flag("disable-features", "UserAgentClientHint")).
		UserAgent(re.ReplaceAllString(userAgent, fmt.Sprintf("Chrome/%s.0.0.0", re.FindStringSubmatch(userAgent)[1])))
}

func Headful() *Chrome {
	return New("").AddFlags(chromedp.Flag("headless", false))
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

func (c *Chrome) Proxy(proxy string) *Chrome {
	c.proxy = proxy
	return c
}

func (c *Chrome) AddFlags(flags ...chromedp.ExecAllocatorOption) *Chrome {
	c.flags = append(c.flags, flags...)
	return c
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

func (c *Chrome) context(ctx context.Context, reset bool) (context.Context, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var new bool
	if c.ctx == nil || (reset && c.Err() != nil) {
		var allocatorCancel, ctxCancel context.CancelFunc
		if c.url == "" {
			opts := chromedp.DefaultExecAllocatorOptions[:]
			if c.useragent != "" {
				opts = append(opts, chromedp.UserAgent(c.useragent))
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

		go func() {
			select {
			case <-c.cancel:
			case <-ctx.Done():
			}
			ctxCancel()
			allocatorCancel()
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

func AddScriptToEvaluateOnNewDocument(script string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) (err error) {
		_, err = page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
		return
	})
}
