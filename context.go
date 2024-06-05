package chrome

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

var _ context.Context = &Chrome{}

func (c *Chrome) Deadline() (deadline time.Time, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, _, _, _ := c.context(context.Background(), false)
	return ctx.Deadline()
}

func (c *Chrome) Done() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, _, _, _ := c.context(context.Background(), false)
	return ctx.Done()
}

func (c *Chrome) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, _, _, err := c.context(context.Background(), false); err != nil {
		return err
	}
	return c.ctx.Err()
}

func (c *Chrome) Value(key any) any {
	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, _, _, _ := c.context(context.Background(), false)
	return ctx.Value(key)
}

func (c *Chrome) context(ctx context.Context, reset bool) (context.Context, context.CancelFunc, bool, error) {
	if c.ctx == nil || (c.ctx != nil && reset && c.ctx.Err() != nil) {
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
		cancel := func() {
			ctxCancel()
			allocatorCancel()
		}
		c.cancel, c.done = make(chan struct{}), make(chan struct{})
		if err := chromedp.Run(c.ctx, c.actions...); err != nil {
			cancel()
			log.Print(err)
			return c.ctx, nil, false, err
		}
		go func() {
			select {
			case <-c.cancel:
			case <-ctx.Done():
			}
			cancel()
			c.cancel = nil
			close(c.done)
		}()
		return c.ctx, cancel, true, nil
	}
	return c.ctx, nil, false, nil
}

func (c *Chrome) newContext(timeout time.Duration) (ctx context.Context, cancel context.CancelFunc, err error) {
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	var ctxCancel context.CancelFunc
	var new bool
	if ctx, ctxCancel, new, err = c.context(ctx, true); err != nil {
		cancel()
		return nil, nil, err
	} else if !new {
		cancel()
		if timeout > 0 {
			var timeoutCancel context.CancelFunc
			ctx, timeoutCancel = context.WithTimeout(c.ctx, timeout)
			ctx, ctxCancel = chromedp.NewContext(ctx, c.ctxOpts...)
			cancel = func() { ctxCancel(); timeoutCancel() }
		} else {
			ctx, cancel = chromedp.NewContext(c.ctx, c.ctxOpts...)
		}
		if err = chromedp.Run(ctx, c.actions...); err != nil {
			cancel()
			return nil, nil, err
		}
	} else {
		cancel = func() { ctxCancel(); cancel() }
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
		if c.done != nil {
			<-c.done
		}
	}
}
