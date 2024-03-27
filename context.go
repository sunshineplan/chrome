package chrome

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"
)

var _ context.Context = &Chrome{}

func (c *Chrome) Deadline() (deadline time.Time, ok bool) {
	c.mu.Lock()
	if c.ctx == nil {
		c.context(context.Background(), false)
	}
	c.mu.Unlock()
	return c.ctx.Deadline()
}

func (c *Chrome) Done() <-chan struct{} {
	c.mu.Lock()
	if c.ctx == nil {
		c.context(context.Background(), false)
	}
	c.mu.Unlock()
	return c.ctx.Done()
}

func (c *Chrome) Err() error {
	c.mu.Lock()
	if c.ctx == nil {
		c.context(context.Background(), false)
	}
	c.mu.Unlock()
	return c.ctx.Err()
}

func (c *Chrome) Value(key any) any {
	c.mu.Lock()
	if c.ctx == nil {
		c.context(context.Background(), false)
	}
	c.mu.Unlock()
	return c.ctx.Value(key)
}

func (c *Chrome) context(ctx context.Context, reset bool) (context.Context, bool) {
	var new bool
	if c.ctx == nil || (c.ctx != nil && reset && c.ctx.Err() != nil) {
		new = true
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
		if err := chromedp.Run(c.ctx, c.actions...); err != nil {
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
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	if _, new := c.context(ctx, true); !new {
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
