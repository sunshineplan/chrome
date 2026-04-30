package chrome

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"
)

// Ensure Chrome implements context.Context interface
var _ context.Context = &Chrome{}

// Deadline returns the browser context's deadline.
func (c *Chrome) Deadline() (time.Time, bool) {
	ctx, _, _, _ := c.context(context.Background(), false)
	return ctx.Deadline()
}

// Done returns a channel that is closed when the browser context is canceled.
func (c *Chrome) Done() <-chan struct{} {
	ctx, _, _, _ := c.context(context.Background(), false)
	return ctx.Done()
}

// Err returns the error that caused the browser context to be canceled.
func (c *Chrome) Err() error {
	ctx, _, _, _ := c.context(context.Background(), false)
	return ctx.Err()
}

// Value retrieves a value from the browser context.
func (c *Chrome) Value(key any) any {
	ctx, _, _, _ := c.context(context.Background(), false)
	return ctx.Value(key)
}

// context initializes or returns the existing browser context.
// It handles both local (exec allocator) and remote browser instances.
// The reset parameter determines whether to create a new context if the current one has ended.
func (c *Chrome) context(ctx context.Context, reset bool) (context.Context, context.CancelFunc, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ctx == nil || (c.ctx != nil && reset && c.ctx.Err() != nil) {
		ctx, cancelCause := context.WithCancelCause(ctx)
		var allocatorCancel context.CancelFunc
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
			if c.enableExtensions {
				opts = append(opts, chromedp.Flag("enable-unsafe-extension-debugging", true))
			} else {
				opts = append(opts, chromedp.Flag("disable-extensions", true))
			}
			ctx, allocatorCancel = chromedp.NewExecAllocator(ctx, append(opts, c.flags...)...)
		} else {
			ctx, allocatorCancel = chromedp.NewRemoteAllocator(ctx, c.url)
		}
		var ctxCancel context.CancelFunc
		c.ctx, ctxCancel = chromedp.NewContext(ctx, c.ctxOpts...)
		cancel := func() {
			cancelCause(nil)
			ctxCancel()
			allocatorCancel()
		}
		c.cancel, c.done = make(chan struct{}), make(chan struct{})
		go func() {
			select {
			case <-c.cancel:
			case <-ctx.Done():
			}
			cancel()
			close(c.done)
		}()
		if err := chromedp.Run(c.ctx, c.actions...); err != nil {
			cancelCause(err)
			ctxCancel()
			allocatorCancel()
			return c.ctx, nil, false, err
		}
		return c.ctx, cancel, true, nil
	}
	return c.ctx, nil, false, nil
}

// newContext creates a new browser context with optional timeout.
// It reuses the existing browser session and creates a new context within it.
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

// NewContext creates a new browser context without timeout.
func (c *Chrome) NewContext() (context.Context, context.CancelFunc, error) {
	return c.newContext(0)
}

// WithTimeout creates a new browser context with the specified timeout duration.
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
